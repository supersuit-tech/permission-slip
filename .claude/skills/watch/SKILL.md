---
name: watch
description: Poll a GitHub PR for new comments and PR reviews and act on them autonomously. Use when the user wants to monitor a PR for feedback and have Claude implement requested changes automatically.
disable-model-invocation: true
argument-hint: "<PR_URL>"
---

# Watch PR for Comments and Reviews

Poll a GitHub Pull Request for all comments (general and inline review comments) and PR reviews (top-level review submissions) and autonomously act on each one.

## Setup

Extract the PR number from the provided URL: `$ARGUMENTS`

Parse the PR number from the URL (e.g., `https://github.com/supersuit-tech/permission-slip/pull/123` → `123`).

Set these variables for the session:
- `PR_NUMBER` — the extracted PR number
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`

## Pre-Poll: Merge from Main

Before entering the polling loop, merge the latest main into the branch. This ensures the branch starts from a clean, up-to-date state before processing review comments.

```bash
git fetch origin main
git merge origin/main --no-edit
```

If the merge produces conflicts, follow the same conflict resolution procedure described in Polling Loop step 2. Run tests and build after resolving.

CI is manual-only (`workflow_dispatch`), so there are no check runs to inspect at this point. CI will be triggered once at the end of the watch session (see Post-Poll step 11).

## Pre-Poll: Process PR Body Checklist

Before entering the polling loop, fetch the PR body and look for unchecked checklist items that Claude Code can address.

### Fetch the PR body

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}" --jq '.body'
```

### Identify unchecked items

Parse the PR body for unchecked checklist items (`- [ ]`). These may appear under any heading, but pay special attention to items under headings like `### Claude Code`, `### Automated`, or similar sections that indicate tasks meant for Claude Code.

**Skip** items that are clearly meant for humans or for OpenClaw (e.g., items under `### OpenClaw`, `### Manual`, or items that require human judgment like "get stakeholder sign-off", "manually verify in production", "design review").

### Act on each item

For each unchecked item that Claude Code can address:

1. **Read and understand** the checklist item.
2. **Implement** the requested change — this might be adding tests, fixing lint issues, updating documentation, running checks, adding error handling, etc.
3. **Run relevant tests** to verify the change doesn't break anything.
4. **Commit** with a clear message referencing the checklist item.
5. **Check off the item** in the PR body by updating it via the API:

```bash
# Fetch current body, update the checkbox, and PATCH it back
CURRENT_BODY=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}" --jq '.body')
# Replace the specific "- [ ] <item text>" with "- [x] <item text>"
UPDATED_BODY=$(echo "$CURRENT_BODY" | sed 's/- \[ \] <exact item text>/- [x] <exact item text>/')
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}" -X PATCH -f body="$UPDATED_BODY"
```

**Important:** Update the PR body after each item (not in batch) to avoid race conditions if the PR body is edited concurrently.

### Push changes

After processing all actionable checklist items, push commits:

```bash
git push -u origin <current-branch>
```

## Polling Loop

Poll every **60 seconds**. On each poll cycle:

### 1. Fetch All Comments and Reviews

Fetch **all** comments and reviews using all three endpoints (PR reviews, review comments, and issue-level comments are separate):

```bash
# PR reviews (top-level review submissions — approve, request changes, comment)
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}/reviews?per_page=100"

# Review comments (inline on diffs)
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/pulls/${PR_NUMBER}/comments?per_page=100"

# Issue-level comments (general PR conversation)
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/issues/${PR_NUMBER}/comments?per_page=100"
```

Handle pagination if there are more than 100 results per endpoint.

**PR reviews note:** Each review object has a `body` (which may be empty) and a `state` (`APPROVED`, `CHANGES_REQUESTED`, `COMMENTED`, `DISMISSED`, `PENDING`). Only process reviews that have a non-empty `body` — empty-body reviews (e.g., a bare approval with no text) have no actionable instructions. Ignore reviews with state `PENDING` (these are drafts not yet submitted).

### 2. Merge from Main

Keep the branch up to date by merging from main on every poll cycle. This prevents the branch from drifting too far from the base branch and avoids large conflict resolutions later.

```bash
# Fetch the latest base branch
git fetch origin main

# Merge main into the current branch
git merge origin/main --no-edit
```

**If the merge succeeds cleanly** (no conflicts and no new commits), continue to the next step — the branch is already up to date.

**If the merge succeeds with new commits**, the branch has been updated. Continue to the next step — tests will be run before pushing.

**If the merge produces conflicts**, resolve them thoughtfully:

1. **Run `git diff --name-only --diff-filter=U`** to list all conflicted files.
2. **For each conflicted file:**
   a. **Read the entire file** to understand the full context — not just the conflict markers.
   b. **Read the PR diff** (`git diff origin/main..HEAD -- <file>`) to understand what this branch intended to change.
   c. **Read the base branch version** (`git show origin/main:<file>`) to understand what changed upstream.
   d. **Understand intent from both sides** — what was the PR trying to accomplish? What did the base branch change introduce? Check recent commit messages on both sides (`git log --oneline HEAD..origin/main -- <file>` and `git log --oneline origin/main..HEAD -- <file>`) for context.
   e. **Resolve the conflict** by editing the file to preserve the intent of both sides. Do NOT blindly accept "ours" or "theirs" — merge the logic correctly so both changes coexist. If the changes are truly incompatible (e.g., both sides renamed the same function differently), prefer the PR branch's version but note this in the decision log.
   f. **Stage the resolved file** with `git add <file>`.
3. **After resolving all files**, complete the merge commit:
   ```bash
   git commit -m "Merge origin/main: resolve conflicts in <list of files>"
   ```
4. **Run tests** (`make test`) and **build** (`make build`) to verify the resolution didn't break anything. If tests fail, investigate and fix before proceeding.
5. **Reset the idle counter to 0** — a merge conflict resolution counts as meaningful activity.

**If the conflict cannot be resolved confidently** (e.g., large-scale structural changes on both sides that require product decisions), do NOT force a resolution. Instead:
- Abort the merge (`git merge --abort`)
- Post a comment on the PR explaining which files conflict and why automatic resolution isn't safe
- Log this in the decision log as an open question

### 3. Track and Process New Comments

- Track the **last-seen ID** for each endpoint separately (review IDs, review comment IDs, and issue comment IDs).
- On the **first poll**, process ALL existing comments and reviews.
- On subsequent polls, only process items with IDs greater than the last-seen ID for that endpoint.
- Process new items in **chronological order** — never skip any, even if multiple arrive between polls.
- Ignore comments and reviews authored by yourself (the bot).
- For PR reviews, use the review `id` field for tracking. Only process reviews with a non-empty `body` and a non-`PENDING` state.

### 4. Act on Each Comment or Review

For each new comment or review:

1. **Read** the instruction in the comment/review body.
2. **Identify** the file and line it's attached to (for inline review comments). For PR reviews, the body applies to the PR as a whole — check the review's `state` for additional context (e.g., `CHANGES_REQUESTED` signals required fixes).
3. **Decide** whether to act on it or disagree:
   - **If you agree**: Implement the change. Commit with a clear message referencing the comment.
   - **If you disagree**: Leave a reply on that comment thread explaining your reasoning.
4. **Do not ask questions** — make your best judgment and implement.
5. **After implementing**: Run relevant tests (`make test-backend` for Go, `make test-frontend` for frontend, `make test` if unsure).

### 5. Resolve Conversations

After addressing a review comment, **resolve the conversation** using the GitHub GraphQL API:

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

To get the thread node ID, use the `node_id` field from the review comment, or query for PR review threads:

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

Match threads by comment `databaseId` to find the correct `id` for resolving.

### 6. Decision Log

For anything you **considered but chose not to do**, or **had questions about**, or **made a judgment call on**:

- Find the GitHub issue linked to the PR (or the PR description itself).
- Append a **checklist** to the issue body under a `## Decision Log` section with entries like:
  - `- [ ] Considered X but chose Y because Z (commit abc123)`
  - `- [ ] Question: Should we also handle edge case X?`

### 7. Process PR Body Checklist

On each poll cycle, re-fetch the PR body and check for any new unchecked items (the PR author or reviewers may add new checklist items between cycles).

Follow the same procedure as the Pre-Poll checklist processing:
1. Fetch the current PR body.
2. Identify unchecked items (`- [ ]`) that Claude Code can address.
3. Skip items meant for humans or OpenClaw.
4. Implement each actionable item, run tests, commit, and check it off in the PR body.

If new checklist items were processed, reset the idle counter to 0.

### 8. Push Changes

After processing all new comments and checklist items in a poll cycle, push your commits:

```bash
git push -u origin <current-branch>
```

### 9. Continue Polling

After each cycle, wait 60 seconds and poll again.

**Idle timeout:** Track the number of consecutive poll cycles with **no new comments, reviews, merge conflicts, or checklist items processed**. If **3 consecutive cycles** pass with no new activity (i.e., 3 minutes of inactivity), **stop polling** and post a wrap-up comment on the PR before exiting (see step 10).

If any cycle finds new comments, reviews, merge conflicts that needed resolution, new commits merged in from main, or unchecked checklist items that were processed, reset the idle counter to 0.

### 10. Post Wrap-Up Comment on Idle Exit

When stopping due to the idle timeout, post a comment on the PR summarizing the entire watch session. Use the following command:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/issues/${PR_NUMBER}/comments" -f body="<comment body>"
```

The comment must include these sections:

**a) Summary of Changes** — A concise overview of all changes made during this watch session. List each commit with its message and what it addressed.

**b) Merge Conflict Resolutions** — If any merge conflicts were resolved during the session, list each one:
- Which files had conflicts
- What the competing changes were (branch vs. base)
- How the conflict was resolved and why

**c) PR Checklist Items** — A summary of checklist items processed from the PR body:
- Which items were completed (checked off) and what was done for each.
- Which items were skipped because they require human action or are meant for OpenClaw.

**d) Decision Log** — A record of key choices made during the session:
- **Implemented:** What review comments were acted on and how.
- **Declined / Disagreed:** Any review comments you chose not to implement, with reasoning.
- **Judgment Calls:** Ambiguous requests where you picked an approach — explain what you chose and why.
- **Open Questions:** Anything that may need human follow-up or further discussion.

Format the comment in markdown. Example structure:

```markdown
## 🤖 Watch Session Summary

### Changes Made
- **`<short description>`** (`<commit hash>`) — <what was changed and why>
- ...

### Merge Conflict Resolutions
- **`<file>`** — <branch changed X, main changed Y> → resolved by <approach> (`<commit hash>`)
- ...

*(Omit this section if no merge conflicts occurred.)*

### PR Checklist Items
- ✅ **`<item text>`** — <what was done> (`<commit hash>`)
- ⏭️ **`<item text>`** — skipped (requires human action / OpenClaw task)
- ...

*(Omit this section if no checklist items were in the PR body.)*

### Decision Log

#### ✅ Implemented
- <comment author> asked for X → implemented in `<commit hash>`
- ...

#### ❌ Declined
- <comment author> suggested X → declined because Y
- ...

#### ⚖️ Judgment Calls
- <description of ambiguous situation> → chose X because Y
- ...

#### ❓ Open Questions
- <anything that needs human follow-up>
- ...

---
*Watch session ended after 3 minutes of inactivity. Processed N comments across M poll cycles.*
```

If no changes were made during the session (e.g., all comments were already addressed before watching started, or no comments existed), still post the comment noting that no action was needed.

### 11. Post-Poll: Trigger CI and Fix Failing Checks

CI is manual-only, so after posting the wrap-up comment, **trigger CI once** against the final state of the branch, wait for it to complete, and fix any failures.

#### a) Trigger the CI workflow

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh workflow run ci.yml --ref "$(git branch --show-current)"
```

#### b) Wait for the run to appear and complete

Poll until the run triggered above finishes. First, wait ~5 seconds for the run to register, then poll:

```bash
# Find the most recent run on this branch
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh run list --workflow=ci.yml --branch "$(git branch --show-current)" --limit 1 --json databaseId,status,conclusion
```

Poll every 30 seconds until `status` is `completed`.

#### c) Check results

If `conclusion` is `success`, no action needed — exit cleanly.

If `conclusion` is `failure`:

1. Fetch the failed run's logs:
   ```bash
   GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh run view <run-id> --log-failed
   ```
2. **Read the failure logs** to understand what went wrong.
3. **Reproduce locally** by running the relevant commands (`make test-backend`, `make test-frontend`, `make build`).
4. **Fix the issue** — read surrounding code context, understand the root cause, implement the fix.
5. **Run tests and build locally** to verify the fix.
6. **Commit** with a clear message (e.g., `fix: resolve failing CI — <brief description>`).
7. **Push** to the branch.
8. **Trigger CI again** (repeat from step a) to verify the fix. If it fails again, repeat the fix cycle up to 3 times before posting a comment that CI cannot be fixed automatically.

If fixes were pushed, append an addendum to the wrap-up comment (post a new comment) noting the additional fixes made:

```markdown
## 🔧 Post-Session Check Fixes

Fixed failing checks after watch session ended:
- **`<description>`** (`<commit hash>`) — <what was failing and how it was fixed>
```

### 12. Trigger Webhook Notification

After all polling, wrap-up, and check fixes are complete, trigger the webhook workflow to notify that the watch session has finished:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh workflow run trigger-webhook.yml -f pr_url="$ARGUMENTS"
```

`$ARGUMENTS` is the original PR URL passed to the `/watch` command. This fires the `trigger-webhook.yml` workflow in the `supersuit-tech/permission-slip` repo, which sends a webhook notification with the PR URL.

This step runs unconditionally — whether or not changes were made during the session.

## Important Rules

- **Never ask for human input** — decide and implement autonomously.
- **Process ALL comments and reviews** — don't skip any, even if multiple arrive between polls.
- **Commit frequently** — one commit per comment or logical group of related comments.
- **Run tests** before pushing to make sure nothing is broken.
- **Run `make build`** before pushing to catch TypeScript compilation errors.
- **Be thorough** — read surrounding code context before making changes.
