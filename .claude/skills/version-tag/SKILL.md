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

Use the GitHub CLI to check whether the tag already exists on the remote:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
  gh api repos/supersuit-tech/permission-slip/git/ref/tags/<package>/v<version> 2>/dev/null
```

If the command succeeds (exit code 0), the tag already exists — print an error and stop. Do NOT overwrite existing tags.

### 3. Resolve Latest Main SHA

Fetch the SHA of the tip of `main` from GitHub without checking out anything locally:

```bash
SHA=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
  gh api repos/supersuit-tech/permission-slip/git/ref/heads/main \
  --jq '.object.sha')
```

If the command fails or `SHA` is empty, print an error and stop.

### 4. Create and Push the Tag

Create the tag reference on GitHub directly via the API — no local branch switching or `git push` needed:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
  gh api repos/supersuit-tech/permission-slip/git/refs \
  -f ref="refs/tags/<package>/v<version>" \
  -f sha="$SHA"
```

If the command fails, print the error output and stop.

### 5. Confirm

Print a summary:
```
Tagged and pushed: <package>/v<version> -> <SHA>
```
