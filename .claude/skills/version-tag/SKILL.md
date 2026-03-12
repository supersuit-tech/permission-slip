---
name: version-tag
description: Create and push a version tag for a package (mobile, cli, or web) from the latest main branch.
argument-hint: <package> <version> (e.g., cli 0.1.0)
---

# Version Tag

Create and push a git tag for a specific package from the latest `main` branch.

## Arguments

This skill takes two arguments:
1. **package** — one of `mobile`, `cli`, or `web`
2. **version** — a semver version string (e.g., `0.1.0`, `1.2.3`)

Example usage: `/version-tag cli 0.1.0`

## Steps

### 1. Parse and Validate Arguments

Parse the two arguments from the input. Validate:
- **package** must be one of: `mobile`, `cli`, `web`. If not, print an error and stop.
- **version** must match semver format (e.g., `0.1.0`, `1.2.3`, `0.0.1-beta.1`). If not, print an error and stop.

Construct the tag name: `<package>/v<version>` (e.g., `cli/v0.1.0`).

### 2. Check for Existing Tag

```bash
git tag -l "<package>/v<version>"
```

If the tag already exists locally or on the remote, print an error and stop. Do NOT overwrite existing tags.

Also check the remote:
```bash
git ls-remote --tags origin "refs/tags/<package>/v<version>"
```

### 3. Stash or Guard Current Work

Before switching branches, check if there are uncommitted changes:
```bash
git status --porcelain
```

If there are changes, stash them:
```bash
git stash push -m "version-tag: stash before tagging <package>/v<version>"
```

Remember to restore the stash and return to the original branch afterward.

### 4. Checkout Main and Pull Latest

```bash
ORIGINAL_BRANCH=$(git branch --show-current)
git checkout main
git pull origin main
```

### 5. Create and Push the Tag

```bash
git tag <package>/v<version>
git push origin <package>/v<version>
```

### 6. Return to Original Branch

```bash
git checkout $ORIGINAL_BRANCH
```

If changes were stashed in step 3, restore them:
```bash
git stash pop
```

### 7. Confirm

Print a summary:
```
Tagged and pushed: <package>/v<version>
```
