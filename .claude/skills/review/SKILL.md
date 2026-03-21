---
name: review
description: Perform a comprehensive, multi-round code review on a GitHub PR. Leaves inline review comments and summaries across up to 6 rounds, with 5-minute waits between rounds to let the author push fixes.
argument-hint: "<PR_URL> [--max-turns <N>]"
---

# Review PR — Multi-Round Comprehensive Code Review

Performs up to 6 rounds of thorough code review on a GitHub Pull Request. Each round submits a PR review with inline comments covering security, architecture, maintainability, code quality, documentation, and performance. Between rounds, waits 5 minutes and re-fetches the diff to check for fixes before reviewing again.

This skill is **strictly read-only** — it never modifies code, creates branches, or makes commits.

## Setup

Parse the arguments from: `$ARGUMENTS`

Extract the PR URL and any flags. The format is: `<PR_URL> [--max-turns <N>]`

The PR URL is **optional**. If not provided, detect it automatically from the current branch:

```bash
PR_URL=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh pr view --json url --jq '.url' 2>/dev/null || echo "")
```

If auto-detection fails (no PR exists for the current branch), abort with a clear error message.

Parse the PR number from the URL (e.g., `https://github.com/supersuit-tech/permission-slip/pull/123` → `123`).

Set these variables for the session:
- `PR_URL` — the PR URL (from arguments or auto-detected)
- `PR_NUMBER` — the extracted PR number
- `MAX_TURNS` — the value of `--max-turns` if passed, `6` otherwise. Must be a positive integer; abort with an error for `0`, negative, or non-numeric values.
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`
- `GH_REPO_PATH` — `supersuit-tech/permission-slip` (used in `gh api` endpoint paths to avoid hardcoding)
- `LAST_HEAD_SHA` — empty string initially, updated after each round
- `ROUND` — current round number, starting at 1

## Step 1: Gather Context

Fetch all the information needed to review the PR.

### 1a. PR Metadata

```bash
$GH_CMD pr view $PR_NUMBER --json title,body,author,labels,headRefOid,baseRefName,files,commits,state
```

Capture:
- `PR_TITLE` — the PR title
- `PR_BODY` — the PR description/body
- `PR_AUTHOR` — who opened the PR
- `HEAD_SHA` — the current head commit SHA
- `CHANGED_FILES` — list of changed file paths
- `PR_STATE` — OPEN, CLOSED, or MERGED

If `PR_STATE` is not `OPEN`, abort — no point reviewing a merged/closed PR.

### 1b. Full Diff

```bash
$GH_CMD pr diff $PR_NUMBER
```

Read and understand the complete diff.

### 1c. Read Changed Files

For each file in `CHANGED_FILES`:
- If the file still exists (not deleted), read it to understand the broader context — not just the diff hunks. **If a file exceeds 500 lines**, read only the diff hunks plus ±30 lines of surrounding context instead of the full file.
- If the file was deleted, only review the deletion in the diff.
- Skip auto-generated files (e.g., `frontend/src/api/schema.d.ts`, vendor directories, lock files).

For large PRs (50+ files), prioritize:
1. Non-test source files (Go, TypeScript, SQL migrations)
2. Test files
3. Config files
4. Generated/vendor files (lowest priority — mention in summary if skipped)

Set `LAST_HEAD_SHA` to `HEAD_SHA`.

## Step 2: Review Loop

Repeat for `ROUND` = 1 to `MAX_TURNS`:

### 2a. Re-fetch (Rounds 2+)

If `ROUND > 1`:

1. **Wait 5 minutes** using the Bash tool with `run_in_background: true`:
   ```bash
   sleep 300 && echo "WAIT_COMPLETE"
   ```
   Set `run_in_background: true` on this Bash call so the agent is not blocked. You will be notified when the sleep completes. While waiting, do NOT proceed to the next step — wait for the background task notification before continuing.

2. **Check PR state:**
   ```bash
   $GH_CMD pr view $PR_NUMBER --json state,headRefOid --jq '{state: .state, head: .headRefOid}'
   ```
   - If PR is merged or closed → go to Step 3 (Final Summary) with a note that the PR was merged/closed.

3. **Check for new commits:**
   - Compare the new `HEAD_SHA` against `LAST_HEAD_SHA`.
   - If they differ, new commits were pushed. Re-fetch the diff and re-read any files that changed:
     ```bash
     $GH_CMD pr diff $PR_NUMBER
     ```
     ```bash
     $GH_CMD pr view $PR_NUMBER --json commits --jq '.commits[] | select(.oid != "") | "\(.oid[:7]) \(.messageHeadline)"'
     ```
   - **Fetch your prior review comments** to avoid posting duplicates:
     ```bash
     $GH_CMD api repos/${GH_REPO_PATH}/pulls/${PR_NUMBER}/comments --jq '[.[] | select(.user.login == "YOUR_BOT_LOGIN") | {path: .path, line: .line, body: .body}]'
     ```
     Track these in a `PRIOR_COMMENTS` list and skip any finding that matches an already-posted comment (same file, same line, same issue).
   - Note which prior review comments may have been addressed by the new commits.
   - Update `LAST_HEAD_SHA`.

### 2b. Perform Comprehensive Review

Review the PR diff and source files across **all** of these dimensions:

**Security**
- Injection vulnerabilities (SQL, XSS, command injection)
- Authentication/authorization gaps
- Secrets or credentials in code
- Race conditions and TOCTOU bugs
- Unsafe deserialization, unsafe type assertions
- Go: unchecked errors, goroutine leaks, context cancellation
- Frontend: `dangerouslySetInnerHTML`, exposed API keys, insecure token storage

**Architecture & Correctness**
- Wrong abstraction layer, coupling issues
- Logic bugs, off-by-one errors, nil pointer risks
- Missing error handling at system boundaries
- Incorrect use of APIs or libraries

**Maintainability**
- DRY violations (duplicated logic that should be extracted)
- Files that are too large and should be split
- Poor naming (variables, functions, types)
- Missing type safety (`any`, unsafe casts without justification)
- Codebase consistency (does this follow established patterns?)

**Code Quality**
- Test coverage gaps for new logic
- Test race conditions (shared state, parallel test issues)
- Edge cases not handled
- Best practices for the language/framework

**Documentation**
- Missing or outdated code comments on non-obvious logic
- README updates needed for new features/changed setup
- OpenAPI spec accuracy if API endpoints were added/changed
- Stale TODO comments

**Performance**
- N+1 queries
- Unnecessary re-renders (React)
- Large allocations in hot paths
- Missing pagination or unbounded queries

**Migration Safety** (if applicable)
- Irreversible schema changes
- Missing rollback path
- Data loss risk
- Locking issues on large tables

**For Rounds 2+, additionally check:**
- Were issues flagged in prior rounds fixed?
- Did fixes introduce new problems?
- Anything missed in earlier rounds that stands out now with fresh eyes?

### 2c. Collect Findings

For each issue found, create an inline comment with:
- **Category prefix:** `**[Security]**`, `**[Architecture]**`, `**[Maintainability]**`, `**[Code Quality]**`, `**[Documentation]**`, `**[Performance]**`, or `**[Migration]**`
- **The issue:** what's wrong or could be improved
- **Suggestion:** how to fix it (be specific — suggest code when possible)

Only flag **real issues**. Do not invent problems or pad the review. If you can't find issues in a category, that's fine — skip it. Quality over quantity.

### 2d. Submit Review

If there are findings, submit a PR review with inline comments:

```bash
echo "$REVIEW_JSON" | $GH_CMD api \
  repos/${GH_REPO_PATH}/pulls/${PR_NUMBER}/reviews \
  --input - || {
    echo "ERROR: Failed to submit Round ${ROUND} review."
    # Log the failure and abort — do not silently continue to the next round.
    exit 1
  }
```

Where `$REVIEW_JSON` is:

```json
{
  "event": "COMMENT",
  "body": "<round summary>",
  "comments": [
    {
      "path": "path/to/file.go",
      "line": 42,
      "side": "RIGHT",
      "body": "**[Security]** This query uses string interpolation which is vulnerable to SQL injection. Use parameterized queries instead:\n```go\ndb.Query(\"SELECT * FROM users WHERE id = $1\", userID)\n```"
    }
  ]
}
```

Build the JSON payload using `jq` to ensure valid JSON. The `line` field is the line number in the file at the PR's HEAD commit (absolute line number, not diff position). The `side` field is **required** — use `"RIGHT"` for additions/modifications (the common case) and `"LEFT"` for deletions on the base side.

**Important:** If a comment targets a line that was not changed in the PR diff, it cannot be an inline comment — include it in the review body instead. The GitHub API only allows inline comments on lines that appear in the diff.

**Round summary body format:**

For Round 1:
```markdown
## Review Round 1/{MAX_TURNS}

**Findings:** {count} inline comments

{paragraph summary of findings — what are the main themes/concerns}

### What Looks Good
{brief acknowledgment of things done well — good patterns, thorough tests, clean code}

---
*Automated review by `/review` — Round 1 of {MAX_TURNS}*
```

For Rounds 2+:
```markdown
## Review Round {N}/{MAX_TURNS}

**Findings:** {count} inline comments

### Changes Since Last Round
{list of new commits if any, and which prior comments they addressed}

### Remaining Issues
{summary of issues that still need attention}

### What Looks Good
{acknowledgment of fixes applied and good patterns}

---
*Automated review by `/review` — Round {N} of {MAX_TURNS}*
```

If there are **no findings** in a round:

```markdown
## Review Round {N}/{MAX_TURNS}

No new issues found.

{If rounds 2+: "All prior comments have been addressed." or "No new commits since last round — no additional issues found on re-examination."}

---
*Automated review by `/review` — Round {N} of {MAX_TURNS}*
```

### 2e. Early Exit Check

After submitting the review, check whether to continue:

- If this round had **no findings** → exit early, go to Step 3. There is no reason to wait and re-review a clean PR.
- If `ROUND >= MAX_TURNS` → exit, go to Step 3.
- Otherwise → increment `ROUND`, loop back to 2a.

## Step 3: Final Summary

After all rounds complete (or early exit), post a final issue comment summarizing the entire review:

```bash
$GH_CMD api \
  repos/${GH_REPO_PATH}/issues/${PR_NUMBER}/comments \
  -f body="$SUMMARY"
```

Summary format:

```markdown
## Code Review Complete

**PR:** #{PR_NUMBER} — {PR_TITLE}
**Author:** @{PR_AUTHOR}
**Rounds completed:** {ROUND}/{MAX_TURNS}
**Reason for stopping:** {completed all rounds | no issues remaining | PR merged | PR closed | max turns reached}

### Outstanding Issues
{any unresolved concerns from the final round, or "None — this PR looks good to merge."}

### Review History
| Round | Inline Comments | Notes |
|-------|----------------|-------|
| 1 | {count} | Initial review |
| 2 | {count} | {e.g., "3 prior issues fixed, 1 new finding"} |
| ... | ... | ... |

### Overall Assessment
{1-2 paragraph take on the PR: is it ready to merge? what were the biggest concerns? what was done well?}

---
*Review performed by Claude Code `/review` skill*
```

## Important Rules

- **Strictly read-only** — NEVER modify files, create branches, make commits, or push code. This skill only reads and comments.
- **Use `COMMENT` event** — never `APPROVE` or `REQUEST_CHANGES`. The skill provides findings; humans make approval decisions.
- **Don't invent problems** — only flag real, substantive issues. If a round has no findings, say so and move on. Padding reviews with nitpicks erodes trust.
- **Be specific** — every comment should explain what's wrong AND how to fix it. Suggest code when possible.
- **Prefix inline comments** with category tags (`[Security]`, `[Architecture]`, etc.) for scannability.
- **Read source files for context** — not just diff hunks. Many issues (DRY violations, architectural problems) are invisible without file-level context. For files over 500 lines, read diff hunks ±30 lines of context instead of the full file to avoid exhausting the context window.
- **Check diff lines** — inline comments can only be placed on lines that appear in the PR diff. If you need to comment on an unchanged line, include it in the review body instead.
- **Acknowledge good work** — every round summary should include a "What Looks Good" section. Reviews that only criticize are demoralizing.
- **Re-fetch between rounds** — always check for new commits before reviewing again to avoid flagging already-fixed issues.
- **Exit early when clean** — if any round has no findings, stop immediately. Don't wait 5 minutes just to re-review a clean PR.
