---
name: review
description: Perform a comprehensive, multi-round code review on a GitHub PR. Leaves inline review comments prioritized by severity (Critical/High/Medium/Low) across up to 6 rounds, with 5-minute waits between rounds to check for fixes. When the review completes, automatically opens a follow-up GitHub issue for any unresolved recommendations.
argument-hint: "<PR_URL> [--max-turns <N>]"
---

# Review PR — Multi-Round Comprehensive Code Review

Performs up to 6 rounds of thorough code review on a GitHub Pull Request. Each round submits a PR review with inline comments covering security, architecture, maintainability, code quality, documentation, and performance. Between rounds, waits 5 minutes and re-fetches the diff to check for fixes before reviewing again.

This skill never modifies code, creates branches, or makes commits. The only writes it performs on GitHub are review comments, one final summary comment, and — at the end — a follow-up tracking issue for any unresolved non-blocking recommendations (see Step 4).

## Reviewer Mindset

You are a Staff+ Engineer performing this review. Follow these principles:

- **Brutally honest, genuinely kind.** Flag real problems without hedging, but frame feedback constructively. Never condescend.
- **Prioritize by real impact.** Every finding gets a severity: Critical > High > Medium > Low. Lead with what matters most.
- **Explain WHY.** Don't just say what's wrong — explain the consequence if it ships. Then give a concrete fix with code.
- **No fluff.** If a category has no issues, skip it. Don't pad reviews to seem thorough.
- **Acknowledge good work.** Reviews that only criticize are demoralizing. Call out good patterns.

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
- `LAST_ACTIVITY_TIME` — timestamp of the most recent new commit detection (or session start). Used for inactivity timeout.
- `ROUND` — current round number, starting at 1
- `CONSECUTIVE_CLEAN_ROUNDS` — number of consecutive rounds with no findings and no new commits. Starts at 0. Reset to 0 whenever new commits are detected or new findings are posted.

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
   - If they differ, new commits were pushed. **Update `LAST_ACTIVITY_TIME`** to the current time. Re-fetch the diff and re-read any files that changed:
     ```bash
     $GH_CMD pr diff $PR_NUMBER
     ```
     ```bash
     $GH_CMD pr view $PR_NUMBER --json commits --jq '.commits[] | select(.oid != "") | "\(.oid[:7]) \(.messageHeadline)"'
     ```
   - **Fetch your prior review comments** to avoid posting duplicates. First, resolve your own login:
     ```bash
     MY_LOGIN=$($GH_CMD api user --jq '.login')
     ```
     Then fetch and filter:
     ```bash
     $GH_CMD api --paginate repos/${GH_REPO_PATH}/pulls/${PR_NUMBER}/comments --jq "[.[] | select(.user.login == \"$MY_LOGIN\") | {path: .path, line: .line, body: .body}]"
     ```
     Track these in a `PRIOR_COMMENTS` list and skip any finding that matches an already-posted comment (same file, same line, same issue).
   - Note which prior review comments may have been addressed by the new commits.
   - **Update `LAST_HEAD_SHA`** to the new `HEAD_SHA` value from step 2a.2 (the `head` field from the state check). This must happen regardless of whether commits changed — the state check fetches the authoritative SHA.

4. **Check inactivity timeout:**
   - If the time elapsed since `LAST_ACTIVITY_TIME` exceeds **10 minutes**, exit to Step 3 with reason "no activity for 10 minutes". This catches the case where the author has stopped responding even though issues remain open.

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

**Severity Guide — assign one to every finding:**

| Severity | Meaning | Examples |
|----------|---------|----------|
| **Critical** | Must fix before merge | Security vulnerabilities, data loss, correctness bugs that will hit production |
| **High** | Should fix before merge | Architectural issues, missing error handling at boundaries, race conditions |
| **Medium** | Improve if time permits | DRY violations, naming issues, missing tests for edge cases |
| **Low** | Optional/nitpick | Style preferences, minor documentation gaps, suggestions for future improvement |

### 2c. Collect Findings

For each issue found, create an inline comment with:
- **Dual prefix:** `**[Severity · Category]**` — e.g., `**[Critical · Security]**`, `**[High · Architecture]**`, `**[Medium · Maintainability]**`, `**[Low · Documentation]**`
- **The issue:** what's wrong
- **Why it matters:** the consequence if it ships (1 sentence)
- **Fix:** concrete suggestion with code when possible

Only flag **real issues**. Do not invent problems or pad the review. If you can't find issues in a category, that's fine — skip it. Quality over quantity.

### 2d. Submit Review

**Only submit a PR review when there are findings.** If the round produced zero inline comments, skip the API call entirely — do not post an empty review. Record the clean round locally (for the final summary's Review History table and the `CONSECUTIVE_CLEAN_ROUNDS` counter) and proceed to Step 2e.

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
      "body": "**[Critical · Security]** This query uses string interpolation which is vulnerable to SQL injection.\n\n**Why:** An attacker can inject arbitrary SQL to read/modify/delete data.\n\n**Fix:** Use parameterized queries:\n```go\ndb.Query(\"SELECT * FROM users WHERE id = $1\", userID)\n```"
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

**Findings:** {count} ({X} critical, {Y} high, {Z} medium, {W} low)

{paragraph summary of findings — what are the main themes/concerns, leading with the highest-severity items}

### What Looks Good
{brief acknowledgment of things done well — good patterns, thorough tests, clean code}

---
*Automated review by `/review` — Round 1 of {MAX_TURNS}*
```

For Rounds 2+:
```markdown
## Review Round {N}/{MAX_TURNS}

**Findings:** {count} ({X} critical, {Y} high, {Z} medium, {W} low)

### Changes Since Last Round
{list of new commits if any, and which prior comments they addressed}

### Remaining Issues
{summary of issues that still need attention, ordered by severity}

### What Looks Good
{acknowledgment of fixes applied and good patterns}

---
*Automated review by `/review` — Round {N} of {MAX_TURNS}*
```

If there are **no findings** in a round, do not post a review for that round (see Step 2d). The clean round is still reflected in the final summary's Review History table.

### 2e. Early Exit Check

After the round completes (whether a review was submitted or skipped), update `CONSECUTIVE_CLEAN_ROUNDS`:
- If this round had **no findings** and **no new commits** since the previous round → increment `CONSECUTIVE_CLEAN_ROUNDS`.
- Otherwise (new findings were posted OR new commits were detected) → reset `CONSECUTIVE_CLEAN_ROUNDS` to 0.

Then check whether to continue:

- If `CONSECUTIVE_CLEAN_ROUNDS >= 3` → exit, go to Step 3 with reason "3 consecutive clean rounds with no new findings or commits".
- If `ROUND >= MAX_TURNS` → exit, go to Step 3.
- If time since `LAST_ACTIVITY_TIME` exceeds **10 minutes** → exit, go to Step 3 with reason "no activity for 10 minutes".
- Otherwise → increment `ROUND`, loop back to 2a.

## Step 3: Final Summary

After all rounds complete (or early exit), post a final issue comment summarizing the entire review:

Build the summary as a shell variable, then post it using a heredoc to avoid quoting issues with special characters in PR titles or author names:

```bash
$GH_CMD api \
  repos/${GH_REPO_PATH}/issues/${PR_NUMBER}/comments \
  --input - <<EOF
{"body": $(echo "$SUMMARY" | jq -Rs .)}
EOF
```

Summary format:

```markdown
## Code Review Complete

**PR:** #{PR_NUMBER} — {PR_TITLE}
**Author:** @{PR_AUTHOR}
**Rounds completed:** {ROUND}/{MAX_TURNS}
**Reason for stopping:** {completed all rounds | 3 consecutive clean rounds | PR merged | PR closed | max turns reached | no activity for 10 minutes}

### Outstanding Issues
{any unresolved concerns from the final round, or "None — this PR looks good to merge."}

### Follow-up Issue
{link to the follow-up issue created in Step 4, or "None — no follow-up recommendations."}

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

**Important:** Post the final summary *after* Step 4 completes so the "Follow-up Issue" section can include the real issue URL (or "None"). If Step 4 is skipped because there are no follow-ups, render that section as "None — no follow-up recommendations."

## Step 4: Create Follow-up Issue for Unresolved Recommendations

After the review loop ends, collect every finding that is still unresolved and is suitable as a follow-up (i.e., not already fixed by the author and not blocking merge). These typically include:

- **Low** severity findings that the author chose not to address.
- **Medium** severity findings that remain unresolved and are reasonable to defer (refactors, DRY cleanups, additional test coverage, non-critical docs).
- Any "future improvement" / "nice to have" suggestions raised during the review.

**Exclude:**
- Critical and High severity findings — these should block merge, not be deferred. If any remain unresolved, mention them in the final summary's Outstanding Issues section but do NOT move them into the follow-up issue.
- Findings that were addressed by a later commit in the PR.
- Purely stylistic nitpicks that don't warrant tracking.

If there are **zero** qualifying follow-up recommendations, skip this step entirely and render the "Follow-up Issue" section of the final summary as "None — no follow-up recommendations."

### 4a. Check for Duplicate Issues

Before creating a new follow-up issue, search open issues to avoid creating duplicates (common when re-running `/review` on the same PR, or when the same underlying concern has been flagged across multiple PRs).

**Check 1 — exact PR match:** Look for an existing open follow-up issue for this same PR.

```bash
EXISTING_FOLLOWUP=$($GH_CMD issue list \
  --state open \
  --search "Follow-ups from #${PR_NUMBER} review in:title" \
  --json number,url \
  --jq '.[0]')
```

If `EXISTING_FOLLOWUP` is non-empty, **skip creating a new issue**. Render the final summary's "Follow-up Issue" section as:

> Existing follow-up issue already open for this PR: {url} — not creating a duplicate. If the current recommendations differ from what's tracked there, update that issue manually.

Also post a short PR comment pointing to the existing issue instead of creating a new one.

**Check 2 — per-recommendation keyword match:** For each recommendation you plan to include, search open issues for semantic duplicates (same file, same concern, or same short-title keywords):

```bash
$GH_CMD issue list \
  --state open \
  --search "<short title keywords> in:title,body" \
  --json number,title,url \
  --limit 3
```

Use judgement when comparing results:
- A match on the same `file.ext:line` reference or clearly overlapping wording → drop that recommendation from the new issue and note the existing issue URL inline in the final summary under Outstanding Issues.
- A loose keyword match with a different subject → keep the recommendation.

If **all** recommendations are dropped as duplicates, skip issue creation entirely and render the "Follow-up Issue" section as:

> None — all non-blocking recommendations are already tracked in existing open issues (see Outstanding Issues for links).

Otherwise, proceed to create the issue with the remaining (non-duplicate) recommendations.

### 4b. Create the Issue

If there is at least one qualifying recommendation remaining after the duplicate check, create a single GitHub issue:

```bash
ISSUE_BODY=$(cat <<'EOF'
Follow-up recommendations from the automated code review of #{PR_NUMBER}.

These items were flagged during review but were considered non-blocking and deferred for a future change. Each item is independently actionable — feel free to tackle them in separate PRs or combine related ones.

## Recommendations

### Medium
- [ ] {short title} — {file.ext}:{line} · {1-sentence description and rationale}
- [ ] ...

### Low
- [ ] {short title} — {file.ext}:{line} · {1-sentence description and rationale}
- [ ] ...

## Context
- Source PR: #{PR_NUMBER} — {PR_TITLE}
- Author: @{PR_AUTHOR}
- Review rounds: {ROUND}/{MAX_TURNS}

---
*Auto-created by Claude Code `/review` skill*
EOF
)

ISSUE_URL=$($GH_CMD issue create \
  --title "Follow-ups from #${PR_NUMBER} review" \
  --body "$ISSUE_BODY")
```

Notes:
- **Only include severity sections that have items.** If there are no Medium items, omit the `### Medium` heading entirely.
- **Use checklist items** (`- [ ]`) per the repo's GitHub Issues convention so progress can be tracked in the issue.
- **Reference files with `path:line`** so each item is easy to locate in the codebase.
- **Keep the title concise** — `Follow-ups from #{PR_NUMBER} review` is the default; adjust only if the PR has an unusually specific theme.
- **Do not add `Closes #...`** — this is a tracking issue, not a fix.
- Capture `ISSUE_URL` and use it in the final summary's "Follow-up Issue" section. Also include it in a short reply on the PR so the author notices:

```bash
$GH_CMD api \
  repos/${GH_REPO_PATH}/issues/${PR_NUMBER}/comments \
  --input - <<EOF
{"body": "Opened a follow-up issue for non-blocking recommendations from this review: ${ISSUE_URL}"}
EOF
```

If the `gh issue create` call fails, log the error, skip posting the PR comment, and render the "Follow-up Issue" section of the final summary as "Failed to create follow-up issue — see recommendations inline in this review." Do not abort the entire skill on an issue-creation failure; the review comments have already been posted and are the primary deliverable.

## Important Rules

- **No code changes** — NEVER modify files, create branches, make commits, or push code. The only writes performed are PR review comments, the final summary comment, and a follow-up tracking issue (Step 4).
- **Use `COMMENT` event** — never `APPROVE` or `REQUEST_CHANGES`. The skill provides findings; humans make approval decisions.
- **Don't invent problems** — only flag real, substantive issues. If a round has no findings, make a note internally and move on. Padding reviews with nitpicks erodes trust.
- **Be specific** — every comment must explain what's wrong, WHY it matters (the consequence), and how to fix it. Suggest code when possible.
- **Use dual-prefix format** — `**[Severity · Category]**` (e.g., `**[High · Architecture]**`) on every inline comment for scannability. Severity levels: Critical, High, Medium, Low.
- **Read source files for context** — not just diff hunks. Many issues (DRY violations, architectural problems) are invisible without file-level context. For files over 500 lines, read diff hunks ±30 lines of context instead of the full file to avoid exhausting the context window.
- **Check diff lines** — inline comments can only be placed on lines that appear in the PR diff. If you need to comment on an unchanged line, include it in the review body instead.
- **Acknowledge good work** — every round summary should include a "What Looks Good" section. Reviews that only criticize are demoralizing.
- **Re-fetch between rounds** — always check for new commits before reviewing again to avoid flagging already-fixed issues.
- **Exit after 3 consecutive clean rounds** — if 3 rounds in a row have no findings and no new commits, stop. This avoids endlessly re-reviewing a stable PR while still giving authors time to push fixes.
