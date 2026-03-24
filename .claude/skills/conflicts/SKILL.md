---
name: conflicts
description: Continuously pull the latest main branch and resolve merge conflicts automatically. Polls every 90 seconds and exits after 10 consecutive clean cycles.
disable-model-invocation: true
---

# Auto-Merge Main and Resolve Conflicts

Continuously fetch and merge the latest `main` branch into the current branch, resolving any merge conflicts automatically. Polls every **90 seconds** and exits after **10 consecutive cycles with no conflicts**.

## Setup

Set these variables for the session:
- `IDLE_COUNT` — 0 (tracks consecutive conflict-free cycles)
- `MAX_IDLE` — 10 (exit threshold)
- `CYCLE` — 0 (total cycle counter)

Before starting the loop, ensure the working tree is clean. If there are uncommitted changes, commit or stash them before proceeding.

## Merge Loop

Poll every **90 seconds**. On each cycle:

### 1. Fetch and Merge Main

```bash
git fetch origin main
git merge origin/main --no-edit
```

### 2. Handle Merge Result

**If the merge succeeds with no new commits** (already up to date):
- Increment `IDLE_COUNT` by 1.
- Log: `Cycle N: clean (IDLE_COUNT/MAX_IDLE consecutive clean cycles)`

**If the merge succeeds with new commits but no conflicts** (fast-forward or clean merge):
- Reset `IDLE_COUNT` to 0 — new commits merged counts as activity.
- Log: `Cycle N: merged new commits from main (no conflicts)`

**If the merge produces conflicts**, resolve them:

1. **Run `git diff --name-only --diff-filter=U`** to list all conflicted files.
2. **For each conflicted file:**
   a. **Read the entire file** to understand the full context — not just the conflict markers.
   b. **Read the PR diff** (`git diff origin/main..HEAD -- <file>`) to understand what this branch intended to change.
   c. **Read the base branch version** (`git show origin/main:<file>`) to understand what changed upstream.
   d. **Understand intent from both sides** — check recent commit messages on both sides for context:
      ```bash
      git log --oneline HEAD..origin/main -- <file>
      git log --oneline origin/main..HEAD -- <file>
      ```
   e. **Resolve the conflict** by editing the file to preserve the intent of both sides. Do NOT blindly accept "ours" or "theirs" — merge the logic correctly so both changes coexist. If the changes are truly incompatible (e.g., both sides renamed the same function differently), prefer the current branch's version.
   f. **Stage the resolved file** with `git add <file>`.
3. **After resolving all files**, complete the merge commit:
   ```bash
   git commit -m "Merge origin/main: resolve conflicts in <list of files>"
   ```
4. **Run tests** (`make test`) and **build** (`make build`) to verify the resolution didn't break anything. If tests fail, investigate and fix before proceeding.
5. **Reset `IDLE_COUNT` to 0** — conflict resolution counts as activity.
6. **Push** the resolved merge to the remote:
   ```bash
   git push -u origin <current-branch>
   ```
7. Log: `Cycle N: resolved conflicts in <files> and pushed`

**If the conflict cannot be resolved confidently** (e.g., large-scale structural changes on both sides that require product decisions):
- Abort the merge (`git merge --abort`)
- Log the issue and notify the user about which files conflict and why automatic resolution isn't safe
- Still reset `IDLE_COUNT` to 0 (there was activity, even though it couldn't be resolved)

### 3. Check Exit Condition

If `IDLE_COUNT` >= `MAX_IDLE` (10 consecutive clean cycles with no conflicts):
- Log: `No merge conflicts for 10 consecutive cycles (15 minutes). Exiting.`
- Exit the loop.

### 4. Wait and Repeat

Sleep for **90 seconds**, then go back to step 1.

## Post-Loop

After exiting the loop, log a summary:

```
Conflicts skill session complete.
- Total cycles: N
- Conflicts resolved: X
- Clean cycles before exit: 10
```

## Important Rules

- **Never ask for human input** — decide and resolve conflicts autonomously.
- **Run tests and build** after every conflict resolution to ensure nothing is broken.
- **Push after every conflict resolution** so the remote branch stays up to date.
- **Only push when there are resolved conflicts** — don't push on clean cycles.
- **Preserve intent from both sides** when resolving conflicts. Read surrounding context, commit messages, and understand what each side was trying to accomplish.
- **Log every cycle** so there is a clear record of what happened.
