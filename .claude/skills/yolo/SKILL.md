---
name: yolo
description: Autonomously implement a GitHub issue or a free-form prompt end-to-end — do the work, open a PR, and watch it through to merge. Use when the user wants to hand off an issue or task completely.
argument-hint: "<ISSUE_URL|prompt text> [--scope \"<heading>\"]"
---

# YOLO — Autonomous Implementation

Takes a GitHub issue URL **or a free-form prompt**, implements the work described, opens a pull request, and hands off to `/watch` to shepherd it through review to merge.

Optionally accepts a `--scope` flag to limit work to a specific section of the issue (e.g., `--scope "Chunk 1A"` or `--scope "Phase 4"`). When scoped, only the content under that heading is implemented — everything else is ignored. The `--scope` flag is only valid when an issue URL is provided.

## Setup

Parse the arguments from: `$ARGUMENTS`

Determine whether the input is an **issue URL** or a **free-form prompt**:

- **Issue URL mode**: The first argument matches a GitHub issue URL pattern (e.g., `https://github.com/.../issues/N` or just a `#N` issue reference). Extract the URL/number and any `--scope` flag.
- **Prompt mode**: The arguments do NOT start with a GitHub issue URL. Treat the entire `$ARGUMENTS` string (minus any flags) as the task description.

Set these variables:
- `MODE` — either `issue` or `prompt`
- `ISSUE_URL` — the full issue URL (issue mode only, empty in prompt mode)
- `ISSUE_NUMBER` — the extracted issue number (issue mode only, empty in prompt mode)
- `TASK_DESCRIPTION` — the free-form prompt text (prompt mode only, empty in issue mode)
- `SCOPE` — the value of `--scope` if provided, empty string otherwise (issue mode only)
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`

## Step 1: Understand the Work

### Issue Mode

Fetch the issue details:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh issue view "$ISSUE_NUMBER" --json title,body,labels,assignees,milestone
```

Read the issue thoroughly.

#### Scoped Execution

If `SCOPE` is set, find the section of the issue body whose heading matches the scope value. The match is **case-insensitive** and looks for any markdown heading (`#`, `##`, `###`, etc.) that contains the scope text. For example, `--scope "Chunk 1A"` matches `## Chunk 1A`, `### Chunk 1A — Database Layer`, etc.

Extract only the content under that heading (up to the next heading of equal or higher level). This becomes your **entire scope of work** — ignore everything else in the issue body. Only check off checklist items that fall within this section.

If no matching heading is found, abort with a comment on the issue: "Could not find a section matching `<SCOPE>` in the issue body."

### Prompt Mode

The task description IS your scope of work. Read it carefully.

If the prompt is unclear or too vague to implement confidently, tell the user what's ambiguous and stop. Do NOT guess at ambiguous requirements.

### Understanding the Work (Both Modes)

Identify:
- **What needs to be built or changed** — the core deliverable
- **Acceptance criteria** — any checklist items, requirements, or success conditions
- **Scope boundaries** — what's explicitly out of scope or deferred
- **Related files** — any files, endpoints, or components mentioned

If the task is unclear or too vague to implement confidently, post a comment on the issue (issue mode) or tell the user (prompt mode) asking for clarification and stop. Do NOT guess at ambiguous requirements.

## Step 2: Create a Feature Branch

Create a descriptive branch name:

**Issue mode:**
```bash
BRANCH="issue-${ISSUE_NUMBER}-<short-kebab-description>"
git checkout -b "$BRANCH"
```

When `SCOPE` is set, incorporate it into the branch name (e.g., `issue-42-chunk-1a-database-layer`).

**Prompt mode:**
```bash
BRANCH="<short-kebab-description>"
git checkout -b "$BRANCH"
```

Pick a concise, descriptive branch name based on the task (e.g., `add-user-avatars`, `fix-login-redirect`).

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

### Checklist Tracking (Issue Mode Only)

If the issue body contains checklist items (`- [ ] ...`), check them off as you complete each one. When `SCOPE` is set, **only check off items within the scoped section** — do not touch checklist items in other sections:

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

Create the PR. The body format differs based on mode:

**Issue mode** — link to the issue. When `SCOPE` is set, include the scope in the PR title (e.g., "Add user avatars (Chunk 1A)") and use `Part of #N` instead of `Closes #N`:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh pr create \
  --title "<concise PR title>" \
  --body "$(cat <<'PREOF'
## Summary

<1-3 bullet points describing the changes>

<"Closes" if unscoped, "Part of" if scoped> #<ISSUE_NUMBER>

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

**Prompt mode** — no issue to link, so include the original prompt for context:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh pr create \
  --title "<concise PR title>" \
  --body "$(cat <<'PREOF'
## Summary

<1-3 bullet points describing the changes>

> **Task:** <original prompt text, quoted>

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

Invoke the `/watch` skill with `--no-notify` to monitor the PR through review and merge it when ready. Auto-merge is enabled by default, so no flag is needed. The `--no-notify` flag suppresses the webhook notification since `/yolo` is a fully autonomous flow — no human ping needed. Since `/watch` auto-detects the PR from the current branch, you don't need to pass the URL explicitly — just the flags:

```
/watch --no-notify
```

Use the `Skill` tool to invoke the watch skill, passing `--no-notify` as the argument. The watch skill will detect the PR from the current branch automatically.

## Important Rules

- **Never ask for human input during implementation** — make your best judgment and implement.
- **Commit frequently** — small, logical commits with clear messages.
- **Run tests before pushing** — never push broken code.
- **Check off issue checklist items in real time** (issue mode) — don't wait until the end.
- **Link the PR to the issue** (issue mode) — use `Closes #N` in the PR body so the issue auto-closes on merge.
- **Include the original prompt in the PR body** (prompt mode) — so reviewers have context on what was requested.
- **Follow all CLAUDE.md guidelines** — especially around migrations, API types, component architecture, and merge conflict minimization.
- **Post-task review** — before opening the PR, run all five review passes from CLAUDE.md (self-review, senior engineer lens, maintainability, code quality, documentation).
