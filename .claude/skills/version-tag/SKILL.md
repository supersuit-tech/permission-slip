---
name: version-tag
description: Create and push a version tag for a package (mobile, cli, or web) from the latest main branch.
argument-hint: <package> [version] (e.g., "cli 0.1.0" or just "cli")
---

# Version Tag

Create and push a git tag for a specific package from the latest `main` branch.

## Arguments

This skill takes one required and one optional argument:
1. **package** (required) — one of `mobile`, `cli`, or `web`
2. **version** (optional) — a semver version string (e.g., `0.1.0`, `1.2.3`). If omitted, automatically increments the patch version of the latest existing tag for the package.

Example usage:
- `/version-tag cli 0.1.0` — tags `cli/v0.1.0`
- `/version-tag cli` — if latest cli tag is `cli/v0.1.0`, tags `cli/v0.1.1`

## Steps

### 1. Parse and Validate Arguments

Parse the arguments from the input. Validate:
- **package** must be one of: `mobile`, `cli`, `web`. If not, print an error and stop.
- If **version** is provided, it must match semver format (e.g., `0.1.0`, `1.2.3`, `0.0.1-beta.1`). If not, print an error and stop.

#### Auto-increment (when version is omitted)

If no version argument is provided, look up the latest existing tag for the package and increment its patch version:

```bash
LATEST=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
  gh api repos/supersuit-tech/permission-slip/git/matching-refs/tags/<package>/v \
  --jq '[.[].ref | ltrimstr("refs/tags/<package>/v")] | map(select(test("^[0-9]+\\.[0-9]+\\.[0-9]+$"))) | sort_by(split(".") | map(tonumber)) | last')
```

- If no existing tags are found (`LATEST` is empty), default to `0.0.1`.
- Otherwise, split `LATEST` on `.`, increment the third (patch) component by 1, and reassemble. For example, `0.1.0` becomes `0.1.1`.

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
