---
name: yolo
description: Autonomously implement a GitHub issue end-to-end — do the work, open a PR, and watch it through to merge. Use when the user wants to hand off an issue completely.
argument-hint: "<ISSUE_URL>"
---

# YOLO — Autonomous Issue Implementation

Takes a GitHub issue URL, implements the work described in it, opens a pull request, and hands off to `/watch --automerge` to shepherd it through review to merge.

## Setup

Parse the issue URL from: `$ARGUMENTS`

Extract the issue number from the URL (e.g., `https://github.com/supersuit-tech/permission-slip/issues/42` → `42`).

Set these variables:
- `ISSUE_URL` — the full issue URL
- `ISSUE_NUMBER` — the extracted issue number
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`

## Step 1: Fetch and Understand the Issue

Fetch the issue details:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh issue view "$ISSUE_NUMBER" --json title,body,labels,assignees,milestone
```

Read the issue thoroughly. Identify:
- **What needs to be built or changed** — the core deliverable
- **Acceptance criteria** — any checklist items, requirements, or success conditions
- **Scope boundaries** — what's explicitly out of scope or deferred
- **Related files** — any files, endpoints, or components mentioned

If the issue is unclear or too vague to implement confidently, post a comment on the issue asking for clarification and stop. Do NOT guess at ambiguous requirements.

## Step 2: Create a Feature Branch

Create a descriptive branch name based on the issue:

```bash
BRANCH="issue-${ISSUE_NUMBER}-<short-kebab-description>"
git checkout -b "$BRANCH"
```

The branch name should be concise but descriptive (e.g., `issue-42-add-user-avatars`).

## Step 3: Plan and Implement

Before writing code, briefly plan your approach:

1. **Identify affected files** — search the codebase to understand the relevant code.
2. **Determine the implementation strategy** — what files to create/modify, in what order.
3. **Consider edge cases** — error handling, validation, migration needs.

Then implement the changes. Follow these principles:

- **Work incrementally** — commit after each logical unit of work, not in one giant commit.
- **Run tests frequently** — after each significant change, run the relevant test suite.
- **Follow existing patterns** — match the codebase's style, conventions, and architecture.
- **Keep diffs minimal** — only touch lines directly related to the task.
- **Follow CLAUDE.md guidelines** — especially around API layer, component architecture, migrations, and testing.

### Checklist Tracking

If the issue body contains checklist items (`- [ ] ...`), check them off as you complete each one:

```bash
CURRENT_BODY=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/issues/${ISSUE_NUMBER}" --jq '.body')
UPDATED_BODY=$(echo "$CURRENT_BODY" | sed 's/- \[ \] <exact item text>/- [x] <exact item text>/')
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh api "/repos/supersuit-tech/permission-slip/issues/${ISSUE_NUMBER}" -X PATCH -f body="$UPDATED_BODY"
```

Check off items **as you complete them**, not all at the end. This gives real-time progress visibility.

## Step 4: Validate

Before opening the PR, run a full validation pass:

1. **Run relevant tests:**
   - Go changes → `make test-backend`
   - Frontend changes → `make test-frontend`
   - Mobile changes → `make mobile-test`
   - Unsure → `make test`
2. **Run build** → `make build` (catches TypeScript compilation errors)
3. **Self-review** — re-read your diff (`git diff origin/main...HEAD`) and look for:
   - Security issues (injection, XSS, missing auth checks)
   - Missing error handling at system boundaries
   - DRY violations or oversized files that should be split
   - Missing test coverage for new logic

Fix any issues found before proceeding.

## Step 5: Push and Open a Pull Request

Push the branch:

```bash
git push -u origin "$BRANCH"
```

Create the PR, linking it to the issue:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh pr create \
  --title "<concise PR title>" \
  --body "$(cat <<'PREOF'
## Summary

<1-3 bullet points describing the changes>

Closes #<ISSUE_NUMBER>

## Test plan

### Claude Code
- [ ] <items Claude Code can verify autonomously>

### OpenClaw
- [ ] <items requiring human judgment>

https://claude.ai/code
PREOF
)" \
  --head "$BRANCH"
```

Capture the PR URL from the output.

## Step 6: Hand Off to /watch

Invoke the `/watch` skill with `--automerge` to monitor the PR through review and merge it when ready:

```
/watch <PR_URL> --automerge
```

Use the `Skill` tool to invoke the watch skill, passing the PR URL and `--automerge` flag as arguments.

## Important Rules

- **Never ask for human input during implementation** — make your best judgment and implement.
- **Commit frequently** — small, logical commits with clear messages.
- **Run tests before pushing** — never push broken code.
- **Check off issue checklist items in real time** — don't wait until the end.
- **Link the PR to the issue** — use `Closes #N` in the PR body so the issue auto-closes on merge.
- **Follow all CLAUDE.md guidelines** — especially around migrations, API types, component architecture, and merge conflict minimization.
- **Post-task review** — before opening the PR, run all five review passes from CLAUDE.md (self-review, senior engineer lens, maintainability, code quality, documentation).
