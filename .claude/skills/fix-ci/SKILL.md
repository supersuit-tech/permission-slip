---
name: fix-ci
description: Automate the CI fix loop — trigger GitHub Actions (CI + audit), read failure logs, fix locally, push, and repeat until green. Use when CI is red and you want a hands-off remediation loop.
argument-hint: "[BRANCH] [--max-rounds <N>] [--work-dir DIR]"
---

# Fix CI — CI + audit remediation loop

Automates the workflow from PR #742 manually: detect branch, trigger **ci.yml** and **audit.yml**, wait for results, parse failures, fix locally with validation, push, and re-run until both pass or the iteration cap is hit.

## Architecture

```
fix-ci-trigger.sh   — One cycle: trigger both workflows, wait (~10 min max each), write logs on failure
Agent (this skill)  — Parse logs, edit code, run local checks, commit, push, PR bookkeeping
```

## Setup

Parse arguments from: `$ARGUMENTS`

Supported forms:

- (empty) — use current branch
- `my-feature-branch` — `git fetch` and checkout that branch before looping
- `--max-rounds 3` — cap fix iterations (default **5**)
- `--work-dir /path` — state directory for log files (default: `/tmp/fix-ci-<pid>` or a fresh dir under `/tmp`)

Set for the session:

- `BRANCH` — target branch name (after optional checkout)
- `MAX_ROUNDS` — default `5`, override with `--max-rounds <N>` (minimum 1)
- `WORK_DIR` — optional; if unset, create `/tmp/fix-ci-$$` once at start and reuse for all rounds
- `GH_CMD` — `GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh`
- `SKILL_DIR` — `.claude/skills/fix-ci`
- `TRIGGER_SCRIPT` — `"${SKILL_DIR}/fix-ci-trigger.sh"`

### Optional branch argument

If the first token is not a flag and not `--max-rounds` / `--work-dir`, treat it as a branch name:

```bash
git fetch origin "<branch>" && git checkout "<branch>"
```

If checkout fails, stop with a clear error.

If no branch argument, use `git branch --show-current`. If empty (detached HEAD), stop with an error.

## Preconditions

- `gh` authenticated for `supersuit-tech/permission-slip`
- Clean enough tree to commit: uncommitted fixes are fine; if you cannot commit (e.g. merge in progress), stop and tell the user

## Main loop

For `round` from 1 through `MAX_ROUNDS`:

### 1. Record local HEAD (optional sanity check)

```bash
LOCAL_SHA=$(git rev-parse HEAD)
```

### 2. Run the trigger script

```bash
bash "${TRIGGER_SCRIPT}" --work-dir "$WORK_DIR"
```

The script **probes** GitHub first: if the latest completed **ci.yml** and **audit.yml** runs for this branch already target **current `HEAD`** and both **succeeded**, it prints `FIX_CI_COMPLETE` and exits 0 without re-triggering. Otherwise it triggers fresh runs (or returns failure logs from the latest failed run for this SHA).

Capture stdout. Parse:

- **`FIX_CI_COMPLETE`** — CI and audit **success** for this push. Go to **Success** (below).
- **Exit 101 + `CI_FAILED`** — read `CI_LOGS_FILE=...` from output, then read that file.
- **Exit 102 + `AUDIT_FAILED`** — read `AUDIT_LOGS_FILE=...`, then read that file.
- **Exit 1** — configuration or `gh` error; stop and report.

### 3. Diagnose (AI)

From the log file(s), determine the **root cause**. Categories to handle:

- TypeScript / build errors (`make build`)
- Go compile or test failures (`make test-backend`, `go test ./...`)
- Frontend tests / lint (`make test-frontend`, eslint output in CI)
- Mobile (`make mobile-test`) if the failing job is mobile
- Database tests / migrations (`go test ./db/... -v`) when indicated
- Lockfile or dependency drift
- **gosec / audit** findings in **audit** logs

**Stop and ask the user** if:

- The failure is ambiguous, flaky without a clear repro, or needs a product/design decision
- Multiple incompatible fixes are possible and logs do not point to one

Do **not** guess on ambiguous security or behavior changes.

### 4. Fix and validate locally

Apply minimal, focused code changes. Then run **relevant** checks before pushing (from `CLAUDE.md`):

| Change area | Run |
|-------------|-----|
| Go only | `make test-backend` |
| Frontend only | `make test-frontend` |
| Mobile only | `make mobile-test` |
| Migrations / `db/` | `go test ./db/... -v` (needs Postgres) |
| Unsure | `make test` |
| Any compile-sensitive change | `make build` |

### 5. Commit and push

One **atomic** commit per fix round with a descriptive message (e.g. `fix(ci): resolve unused import in handler`).

```bash
git push -u origin "$(git branch --show-current)"
```

### 6. Next round

If not yet at `MAX_ROUNDS`, go to step 2. The trigger script always starts **new** workflow runs after your push.

## Success

When `fix-ci-trigger.sh` exits 0:

1. Print the **CI run URL** and **audit run URL** from script output (`CI_RUN_URL=`, `AUDIT_RUN_URL=`).
2. **Pull request:** If there is no PR for the current branch, create one (same conventions as `/yolo` Step 5 — ready for review, link context in body). If a PR already exists, note its URL.

```bash
PR_URL=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip gh pr view --json url --jq '.url' 2>/dev/null || echo "")
```

If `PR_URL` is empty and the branch is not the default branch (`main`), create the PR with `gh pr create` (non-draft, per `CLAUDE.md`).

End the session with: `Pull request: <url>` when a PR exists (plain text, per `CLAUDE.md`).

## Exhausted rounds

If you reach `MAX_ROUNDS` with CI or audit still failing:

- Summarize what failed, last log hints, and last run URLs
- Suggest manual triage; do not keep looping

## Guardrails

- **Default `MAX_ROUNDS`:** 5
- **No auto-merge** — this skill only fixes CI/audit; it does not merge PRs
- **Minimal diffs** — follow merge-conflict minimization rules in `CLAUDE.md`

## Example invocations

```
/fix-ci
/fix-ci my-feature-branch
/fix-ci --max-rounds 3
/fix-ci cursor/some-branch --max-rounds 8 --work-dir /tmp/fix-ci-session
```
