---
name: watch
description: Poll a GitHub PR for new comments and PR reviews and act on them autonomously. Use when the user wants to monitor a PR for feedback and have Claude implement requested changes automatically.
argument-hint: "[PR_URL] [--no-automerge] [--max-turns <N>] [--no-notify]"
---

# Watch PR for Comments and Reviews

Poll a GitHub Pull Request for all comments (general and inline review comments) and PR reviews (top-level review submissions) and autonomously act on each one.

This skill uses shell scripts for all mechanical bookkeeping (polling, fetching, deduplication, idle tracking, checklist parsing, wrap-up generation) and only invokes the agent for tasks requiring AI reasoning (implementing changes, resolving conflicts, diagnosing CI failures).

## Architecture

```
watch-poll.sh          — Deterministic polling loop (no AI needed)
  ├── Fetches comments from 3 GitHub API endpoints
  ├── Deduplicates by tracking last-seen IDs
  ├── Filters out bot-authored comments
  ├── Merges from main, detects conflicts
  ├── Parses PR body for unchecked Claude Code checklist items
  ├── Writes work-items.json when there's actionable work
  └── Generates wrap-up comment from action-log.json on idle timeout

watch-post.sh          — Post-session tasks (no AI needed)
  ├── Triggers CI and audit workflows
  ├── Waits for completion
  ├── Auto-merges if enabled and checks pass
  └── Triggers webhook notification (unless --skip-webhook)

watch-append-wrapup-ci.sh — Appends CI/audit remediation bullets to the wrap-up PR comment

Agent (this skill)     — Only invoked when reasoning is needed
  ├── Implements review comment requests
  ├── Resolves merge conflicts
  ├── Diagnoses and fixes CI failures
  └── Appends actions to action-log.json
```

## Setup

Parse the arguments from: `$ARGUMENTS`

Extract any PR URL and flags. The format is: `[PR_URL] [--no-automerge] [--max-turns <N>] [--no-notify]`

The PR URL is **optional**. If not provided, detect it automatically from the current branch:

```bash
PR_URL=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh pr view --json url --jq '.url' 2>/dev/null || echo "")
```

If auto-detection fails (no PR exists for the current branch), abort with a clear error message.

Parse the PR number from the URL (e.g., `https://github.com/supersuit-tech/permission-slip/pull/123` → `123`).

Set these variables for the session:
- `PR_URL` — the PR URL (from arguments or auto-detected from current branch)
- `PR_NUMBER` — the extracted PR number
- `AUTO_MERGE` — `true` by default, `false` if `--no-automerge` was passed
- `MAX_TURNS` — the value of `--max-turns` if passed, `0` otherwise (0 = unlimited)
- `NO_NOTIFY` — `true` if `--no-notify` was passed, `false` otherwise
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`
- `SKILL_DIR` — the directory containing this skill file (`.claude/skills/watch`)

Preserve `WORK_DIR` from `watch-poll.sh` session JSON for the whole session — it holds `action-log.json`, the wrap-up comment id file, and other state. The poll script writes `pr-number.txt` and `wrapup-comment-id` when it posts the wrap-up.

## Orchestration Loop

The agent orchestrates the session by running the shell scripts and only doing AI work when the scripts signal that it's needed.

### Step 1: Run the polling script

```bash
# First run (no --work-dir):
bash "${SKILL_DIR}/watch-poll.sh" "${PR_URL}" $([[ "$AUTO_MERGE" == "false" ]] && echo "--no-automerge") $([[ "$MAX_TURNS" -gt 0 ]] && echo "--max-turns $MAX_TURNS") 2>&1
# Capture WORK_DIR from the session context JSON in the output.

# Subsequent runs (reuse WORK_DIR to preserve state):
bash "${SKILL_DIR}/watch-poll.sh" "${PR_URL}" $([[ "$AUTO_MERGE" == "false" ]] && echo "--no-automerge") $([[ "$MAX_TURNS" -gt 0 ]] && echo "--max-turns $MAX_TURNS") --work-dir "$WORK_DIR" 2>&1
```

The script handles:
- Fetching comments from all 3 GitHub API endpoints (reviews, review comments, issue comments)
- Deduplication via last-seen ID tracking per endpoint
- Filtering out bot-authored comments
- Merging from main and detecting conflicts
- Parsing the PR body for unchecked `### Claude Code` checklist items
- Idle counter tracking (6 consecutive empty cycles = timeout)
- Turn limit tracking (exits when `--max-turns` agent invocations reached)
- Generating and posting the wrap-up comment on idle timeout

### Step 2: Check script output

The script communicates via stdout signals and exit codes:

**Exit code 0 + `AGENT_NEEDED`**: The script found actionable work. Read `WORK_ITEMS_FILE` for the structured work items:

```json
{
  "pr_number": "123",
  "pr_url": "https://github.com/.../pull/123",
  "branch": "feature-branch",
  "cycle": 1,
  "comments": [
    {
      "type": "review_comment",
      "id": 456,
      "author": "reviewer",
      "body": "Please rename this variable",
      "path": "src/main.go",
      "line": 42,
      "diff_hunk": "...",
      "node_id": "..."
    }
  ],
  "merge_status": "clean|updated|conflict",
  "conflict_files": ["file1.go", "file2.ts"],
  "checklist_items": [
    {"type": "checklist", "text": "Add unit tests for the new handler"}
  ]
}
```

**Exit code 0 + `IDLE_TIMEOUT`**: The session ended due to inactivity or the `--max-turns` limit being reached. The script already posted the wrap-up comment (and recorded its id under `WORK_DIR` for later edits). Proceed to Step 5 (post-session / CI loop).

**Exit code 100**: Pre-poll merge conflict. The script detected conflicts before the polling loop started. Read `WORK_ITEMS_FILE` for conflict details and resolve them (see Step 3), then re-run the polling script.

### Step 3: Process work items (AI reasoning)

For each item in `work-items.json`, apply the appropriate action:

#### Comments and Reviews

**Default: Parallel processing.** When the `comments` array contains multiple items, process them in parallel using the Agent tool. Each comment gets its own subagent running in an isolated worktree, so they can't conflict with each other during implementation. After all agents complete, cherry-pick their commits onto the current branch, resolve any conflicts, and run tests once.

**Parallel processing flow (2+ comments):**

1. **Group comments by file.** If multiple comments target the same file (same `path`), group them together — they must be handled by a single agent to avoid edit conflicts. Comments on different files (or general PR-level comments) can each get their own agent.
2. **Spawn one Agent per group** using the Agent tool with `isolation: "worktree"`. Each agent receives:
   - The comment(s) it's responsible for (body, path, line, diff_hunk, node_id)
   - The PR number and GH command prefix for resolving threads and reacting
   - Instructions to: read the code, implement the change, commit, react to the comment, reply in the thread, and resolve the conversation
   - The action log format so it can return structured log entries
3. **Launch all agents in a single message** (parallel tool calls) so they run concurrently.
4. **After all agents complete**, collect their results:
   - For each agent that made changes in a worktree, cherry-pick or merge its commits onto the current branch
   - If cherry-picks conflict, resolve the conflicts (prefer the change that matches the reviewer's intent)
   - Collect action log entries from each agent's output
5. **Run tests once** after all changes are integrated (`make test-backend` for Go, `make test-frontend` for frontend, `make test` if unsure).
6. **Append all action log entries** to the action log file.

**Agent prompt template** — each parallel agent should receive a prompt like:

```
You are handling a PR review comment for PR #<PR_NUMBER> on supersuit-tech/permission-slip.

Comment by @<AUTHOR>:
> <BODY>

File: <PATH> (line <LINE>)
Diff context:
<DIFF_HUNK>

Instructions:
1. Read the file and surrounding context to understand the code.
2. Implement the requested change. Commit with a descriptive message.
3. React to the comment and reply in the thread:
   - React with +1: GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api repos/supersuit-tech/permission-slip/pulls/comments/<COMMENT_ID>/reactions -f content="+1"
   - Reply: GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api repos/supersuit-tech/permission-slip/pulls/comments/<COMMENT_ID>/replies -f body="<your reply>"
4. Resolve the review thread using the GraphQL API (thread node_id: <NODE_ID>).
5. Return a JSON action log entry: {"type": "implemented", "author": "<AUTHOR>", "request": "<summary>", "commit": "<hash>"}
   Or if you disagree: {"type": "declined", "author": "<AUTHOR>", "request": "<summary>", "reason": "<explanation>"}
```

**Single comment flow (1 comment) or fallback:** Process inline without spawning a subagent:

1. **Read** the instruction in the comment body.
2. **Identify** the file and line it's attached to (for inline review comments — use `path` and `line` fields). For PR reviews, the body applies to the PR as a whole — check `state` for context (`CHANGES_REQUESTED` signals required fixes). For issue comments (normal PR conversation), treat them as general feedback that may reference specific files or areas of the code.
3. **Decide** whether to act on it or disagree:
   - **If you agree**: Implement the change. Commit with a clear message referencing the comment.
   - **If you disagree**: Leave a reply on that comment thread explaining your reasoning.
4. **Do not ask questions** — make your best judgment and implement.
5. **After implementing**: Run relevant tests (`make test-backend` for Go, `make test-frontend` for frontend, `make test` if unsure).
6. **Resolve the conversation** using the GitHub GraphQL API:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api graphql -f query='
mutation {
  resolveReviewThread(input: {threadId: "THREAD_NODE_ID"}) {
    thread {
      isResolved
    }
  }
}'
```

To get the thread node ID, query for PR review threads and match by `databaseId`:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api graphql -f query='
query {
  repository(owner: "supersuit-tech", name: "permission-slip") {
    pullRequest(number: PR_NUMBER) {
      reviewThreads(first: 100) {
        nodes {
          id
          isResolved
          comments(first: 1) {
            nodes {
              body
              databaseId
            }
          }
        }
      }
    }
  }
}'
```

7. **Log the action** by appending to the action log file (see Action Log Format below).

#### Merge Conflicts

If `merge_status` is `"conflict"` and `conflict_files` is non-empty:

1. **Run `git diff --name-only --diff-filter=U`** to list all conflicted files.
2. **For each conflicted file:**
   a. **Read the entire file** to understand the full context — not just the conflict markers.
   b. **Read the PR diff** (`git diff origin/main..HEAD -- <file>`) to understand what this branch intended to change.
   c. **Read the base branch version** (`git show origin/main:<file>`) to understand what changed upstream.
   d. **Understand intent from both sides** — check recent commit messages on both sides for context.
   e. **Resolve the conflict** by editing the file to preserve the intent of both sides. Do NOT blindly accept "ours" or "theirs". If changes are truly incompatible, prefer the PR branch's version but note this in the action log.
   f. **Stage the resolved file** with `git add <file>`.
3. **Complete the merge commit:**
   ```bash
   git commit -m "Merge origin/main: resolve conflicts in <list of files>"
   ```
4. **Run tests** (`make test`) and **build** (`make build`) to verify the resolution.
5. **Log conflict resolutions** in the action log.

**If the conflict cannot be resolved confidently**, abort the merge (`git merge --abort`), post a comment on the PR explaining why, and log it as an open question.

#### Checklist Items

For each item in the `checklist_items` array:

1. **Read and understand** the checklist item text.
2. **Implement** the requested change (adding tests, fixing lint, updating docs, etc.).
3. **Run relevant tests** to verify.
4. **Commit** with a clear message referencing the checklist item.
5. **Check off the item** in the PR body:
   ```bash
   CURRENT_BODY=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}" --jq '.body')
   UPDATED_BODY=$(echo "$CURRENT_BODY" | sed 's/- \[ \] <exact item text>/- [x] <exact item text>/')
   GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}" -X PATCH -f body="$UPDATED_BODY"
   ```
   Update the PR body after each item to avoid race conditions.
6. **Log the action** in the action log.

#### Decision Log

For anything you **considered but chose not to do**, or **had questions about**, or **made a judgment call on**:

- Find the GitHub issue linked to the PR (or the PR description itself).
- Append a **checklist** to the issue body under a `## Decision Log` section with entries like:
  - `- [ ] Considered X but chose Y because Z (commit abc123)`
  - `- [ ] Question: Should we also handle edge case X?`

#### Push Changes

After processing all work items, push your commits:

```bash
git push -u origin <current-branch>
```

### Step 3b: Update the action log

After processing all work items, append entries to the action log file (`ACTION_LOG_FILE` from the script output). The agent must read the current log, append new entries, and write it back.

#### Action Log Format

The action log is a JSON array. Each entry has a `type` field and type-specific fields:

```json
[
  {
    "type": "change",
    "description": "Rename variable for clarity",
    "detail": "Renamed `usr` to `currentUser` per reviewer request",
    "commit": "abc1234"
  },
  {
    "type": "implemented",
    "author": "reviewer-username",
    "request": "rename the variable",
    "commit": "abc1234"
  },
  {
    "type": "declined",
    "author": "reviewer-username",
    "request": "add caching here",
    "reason": "premature optimization, no performance issue observed"
  },
  {
    "type": "conflict_resolution",
    "file": "src/main.go",
    "detail": "branch added handler, main renamed package — kept both changes",
    "commit": "def5678"
  },
  {
    "type": "checklist_done",
    "text": "Add unit tests for the new handler",
    "detail": "Added 3 test cases covering success, auth failure, and validation",
    "commit": "ghi9012"
  },
  {
    "type": "checklist_skipped",
    "text": "Get stakeholder sign-off",
    "reason": "requires human action / OpenClaw task"
  },
  {
    "type": "judgment",
    "description": "Whether to split the handler file",
    "choice": "split into handler.go and handler_test.go",
    "reason": "file was over 500 lines, splitting improves maintainability"
  },
  {
    "type": "open_question",
    "description": "Should we also add rate limiting to this endpoint?"
  },
  {
    "type": "ci_remediation",
    "workflow": "ci.yml",
    "conclusion": "failure",
    "detail": "go test ./db/... failed: missing migration grant — added GRANT in migration 20260320…",
    "commit": "abc1234"
  },
  {
    "type": "audit_remediation",
    "workflow": "audit.yml",
    "conclusion": "failure",
    "detail": "gosec flagged unsafe usage — refactored to use sanitized input",
    "commit": "def5678"
  },
  {
    "type": "ci_fix_exhausted",
    "workflow": "ci.yml",
    "detail": "Stopped after 15 remediation attempts; integration test still flakes on job X — needs human triage"
  }
]
```

Use **`ci_remediation`** / **`audit_remediation`** after each failed post-session run you fix (one object per fix round is enough; add both if you fixed issues revealed by both workflows in one push). **`ci_fix_exhausted`** is only for hitting the 15-attempt limit. These types appear in the **next** full wrap-up if you re-run poll with a fresh session; within the same session, **`watch-append-wrapup-ci.sh`** appends under a single **## 🔧 CI / audit remediation** heading: the first batch adds that heading plus **### Remediation round 1**, and later batches add **### Remediation round N** only (no duplicate H2).

### Step 4: Loop back to polling

After the agent finishes processing work items, go back to **Step 1** and run the polling script again. The script maintains state across invocations via temp files in its work directory.

**Important**: Pass the same work directory to the script on subsequent runs so it preserves last-seen IDs and the action log. The script's `WORK_DIR` is printed in its session context output — capture it on the first run and reuse it.

### Step 5: Post-session tasks — CI / audit loop until green

When the polling script exits with `IDLE_TIMEOUT`, the wrap-up comment has already been posted. You must then **drive CI to success** (and audit to success) in a loop: fix, push, re-run workflows, repeat. **Do not stop after a single failed post-session run** unless you hit the exhaustion limit below.

**Loop (run until both CI and audit report `success`, or you exhaust retries):**

1. Run post-session. Always pass **`--work-dir "$WORK_DIR"`** (same session directory as `watch-poll.sh`) so CI/audit logs are written under that directory (avoids `/tmp` clashes between concurrent watch sessions) and so the webhook sentinel file is session-scoped. On the **first** iteration of this loop after wrap-up, pass **`--skip-webhook`** so the webhook does not fire until CI is actually green (this creates **`${WORK_DIR}/post-webhook-pending`** on success):
   ```bash
   bash "${SKILL_DIR}/watch-post.sh" "${PR_URL}" --work-dir "$WORK_DIR" $([[ "$AUTO_MERGE" == "false" ]] && echo "--no-automerge") $([[ "$NO_NOTIFY" == "true" ]] && echo "--no-notify") --skip-webhook 2>&1
   ```
   On **subsequent** iterations (after you pushed fixes), omit `--skip-webhook` so each successful pass can notify as usual, **or** keep using `--skip-webhook` until the final successful iteration and then run once with **`--webhook-only`** (see step 2b below).

2. **Exit code 0** — Both workflows concluded `success`. If the webhook was sent this run (no `--skip-webhook`), **`post-webhook-pending`** is removed automatically. If you used **`--skip-webhook`** and `NO_NOTIFY` is false, run once with **`--webhook-only`** — this **requires** `--work-dir` and an existing **`post-webhook-pending`** file (otherwise exit **2**); on success the webhook runs and the sentinel is removed:
   ```bash
   bash "${SKILL_DIR}/watch-post.sh" "${PR_URL}" --work-dir "$WORK_DIR" $([[ "$NO_NOTIFY" == "true" ]] && echo "--no-notify") --webhook-only 2>&1
   ```
   Read merge result from stdout (`PR merged successfully`, etc.).

3. **Exit code 101 (CI)** or **102 (audit)** — This is expected while fixing. **Automatically:**
   - Read logs from `CI_LOGS_FILE` or `AUDIT_LOGS_FILE` in the script output.
   - Diagnose and **implement fixes** (same rigor as review comments): run relevant tests (`make test`, `make build` before push when the failure could be compile-related).
   - **Append to `action-log.json`** at least one object describing what you did (see **CI remediation** types below). Use a real summary in `detail` (root cause + fix), not placeholders.
   - **Push** commits to the PR branch.
   - **Patch the wrap-up comment** with the new remediation entry:
     ```bash
     bash "${SKILL_DIR}/watch-append-wrapup-ci.sh" --work-dir "$WORK_DIR"
     ```
   - **Go back to step 1** and re-run `watch-post.sh` with the same **`--work-dir "$WORK_DIR"`** so log paths stay session-isolated.

4. **Retry limit** — Repeat step 3 **until CI and audit both succeed**, or until **15** failed post-session attempts (101/102) **in this loop** for the same PR session. If you hit the limit, append an `ci_fix_exhausted` entry to `action-log.json`, run `watch-append-wrapup-ci.sh` again, post a short PR comment that automated remediation stopped after 15 attempts and summarize what is still failing, then stop.

`watch-post.sh` treats any workflow **conclusion other than `success`** (including `failure`, `cancelled`, `skipped`, `timed_out`, or **`timeout`** when the wait window expires) as needing remediation — exit **101** for CI, **102** for audit.

**After the script completes successfully (exit code 0) with `AUTO_MERGE=true`:**

Check the script output for the merge result:
- `"[post] PR merged successfully."` → The merge completed. Report it as **merged**, not "attempted".
- `"[post] Auto-merge failed."` → The merge failed. The script already posted a comment on the PR.
- `"[post] Auto-merge enabled but checks did not pass."` → Should not occur on exit 0; if it does, treat as inconsistent and re-run post-session.

**Do NOT hedge** when the script output clearly indicates success or failure.

## Important Rules

- **Never ask for human input** — decide and implement autonomously.
- **Process ALL comments and reviews** — don't skip any, even if multiple arrive between polls.
- **Commit frequently** — one commit per comment or logical group of related comments.
- **Run tests** before pushing to make sure nothing is broken.
- **Run `make build`** before pushing to catch TypeScript compilation errors.
- **Be thorough** — read surrounding code context before making changes.
- **Always update the action log** after completing work — the wrap-up comment is generated from it.
