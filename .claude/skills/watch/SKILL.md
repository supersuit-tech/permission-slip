---
name: watch
description: Poll a GitHub PR for new comments and PR reviews and act on them autonomously. Use when the user wants to monitor a PR for feedback and have Claude implement requested changes automatically.
disable-model-invocation: true
argument-hint: "<PR_URL> [--automerge] [--max-turns <N>]"
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
  └── Triggers webhook notification

Agent (this skill)     — Only invoked when reasoning is needed
  ├── Implements review comment requests
  ├── Resolves merge conflicts
  ├── Diagnoses and fixes CI failures
  └── Appends actions to action-log.json
```

## Setup

Parse the arguments from: `$ARGUMENTS`

Extract the PR URL and any flags. The format is: `<PR_URL> [--automerge] [--max-turns <N>]`

Parse the PR number from the URL (e.g., `https://github.com/supersuit-tech/permission-slip/pull/123` → `123`).

Set these variables for the session:
- `PR_URL` — the PR URL extracted from the arguments
- `PR_NUMBER` — the extracted PR number
- `AUTO_MERGE` — `true` if `--automerge` was passed, `false` otherwise
- `MAX_TURNS` — the value of `--max-turns` if passed, `0` otherwise (0 = unlimited)
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`
- `SKILL_DIR` — the directory containing this skill file (`.claude/skills/watch`)

## Orchestration Loop

The agent orchestrates the session by running the shell scripts and only doing AI work when the scripts signal that it's needed.

### Step 1: Run the polling script

```bash
# First run (no --work-dir):
bash "${SKILL_DIR}/watch-poll.sh" "${PR_URL}" $([[ "$AUTO_MERGE" == "true" ]] && echo "--automerge") $([[ "$MAX_TURNS" -gt 0 ]] && echo "--max-turns $MAX_TURNS") 2>&1
# Capture WORK_DIR from the session context JSON in the output.

# Subsequent runs (reuse WORK_DIR to preserve state):
bash "${SKILL_DIR}/watch-poll.sh" "${PR_URL}" $([[ "$AUTO_MERGE" == "true" ]] && echo "--automerge") $([[ "$MAX_TURNS" -gt 0 ]] && echo "--max-turns $MAX_TURNS") --work-dir "$WORK_DIR" 2>&1
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

**Exit code 0 + `IDLE_TIMEOUT`**: The session ended due to inactivity or the `--max-turns` limit being reached. The script already posted the wrap-up comment. Proceed to Step 4 (post-session).

**Exit code 100**: Pre-poll merge conflict. The script detected conflicts before the polling loop started. Read `WORK_ITEMS_FILE` for conflict details and resolve them (see Step 3), then re-run the polling script.

### Step 3: Process work items (AI reasoning)

For each item in `work-items.json`, apply the appropriate action:

#### Comments and Reviews

For each comment or review in the `comments` array (including `review`, `review_comment`, and `issue_comment` types — normal PR comments often contain actionable feedback just like inline review comments):

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
  }
]
```

### Step 4: Loop back to polling

After the agent finishes processing work items, go back to **Step 1** and run the polling script again. The script maintains state across invocations via temp files in its work directory.

**Important**: Pass the same work directory to the script on subsequent runs so it preserves last-seen IDs and the action log. The script's `WORK_DIR` is printed in its session context output — capture it on the first run and reuse it.

### Step 5: Post-session tasks

When the polling script exits with `IDLE_TIMEOUT`, the wrap-up comment has already been posted by the script. Run the post-session script:

```bash
bash "${SKILL_DIR}/watch-post.sh" "${PR_URL}" $([[ "$AUTO_MERGE" == "true" ]] && echo "--automerge") 2>&1
```

This handles:
- Triggering CI and audit workflows
- Waiting for them to complete
- Auto-merging if enabled and checks pass
- Triggering the webhook notification

**After the script completes successfully (exit code 0) with `AUTO_MERGE=true`:**

Check the script output for the merge result. The script prints clear signals:
- `"[post] PR merged successfully."` → The merge completed. Report it as **merged**, not "attempted". Do NOT say "may need human approval" — if the script printed this, the PR is merged.
- `"[post] Auto-merge failed."` → The merge failed. The script already posted a comment on the PR. Report the failure and suggest the user merge manually.
- `"[post] Auto-merge enabled but checks did not pass."` → CI or audit failed, merge was skipped. Report which checks failed.

**Do NOT hedge or use vague language like "attempted" or "may need human approval" when the script output clearly indicates success or failure.** Read the output and report what actually happened.

**If the post-session script exits with code 101 (CI failure) or 102 (audit failure):**

1. Read the failure logs from the file path in the script output.
2. Reproduce locally by running the relevant commands.
3. Fix the issue, commit, and push.
4. Post an addendum comment:
   ```markdown
   ## 🔧 Post-Session Check Fixes

   Fixed failing checks after watch session ended:
   - **`<description>`** (`<commit hash>`) — <what was failing and how it was fixed>
   ```
5. Re-run the post-session script. Repeat up to 3 times before posting a comment that CI cannot be fixed automatically.

## Important Rules

- **Never ask for human input** — decide and implement autonomously.
- **Process ALL comments and reviews** — don't skip any, even if multiple arrive between polls.
- **Commit frequently** — one commit per comment or logical group of related comments.
- **Run tests** before pushing to make sure nothing is broken.
- **Run `make build`** before pushing to catch TypeScript compilation errors.
- **Be thorough** — read surrounding code context before making changes.
- **Always update the action log** after completing work — the wrap-up comment is generated from it.
