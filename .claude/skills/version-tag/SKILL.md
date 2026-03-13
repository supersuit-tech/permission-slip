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

### 4. Sync package.json Version

Check whether `<package>/package.json` exists in the repo and whether its `version` field matches the target version. If it doesn't match, bump it via the GitHub API (creating a commit directly on `main`) and update `SHA` to the new commit.

```bash
# Fetch the file metadata and content
FILE_RESP=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
  gh api repos/supersuit-tech/permission-slip/contents/<package>/package.json \
  --jq '{sha: .sha, content: .content}')

FILE_SHA=$(echo "$FILE_RESP" | jq -r '.sha')
# Decode content (GitHub returns base64 with newlines)
CURRENT_CONTENT=$(echo "$FILE_RESP" | jq -r '.content' | base64 -d)
PKG_VERSION=$(echo "$CURRENT_CONTENT" | jq -r '.version')

if [ "$PKG_VERSION" != "<version>" ]; then
  echo "package.json version is $PKG_VERSION, bumping to <version>..."

  # Produce updated JSON preserving formatting
  UPDATED_CONTENT=$(echo "$CURRENT_CONTENT" | jq --arg v "<version>" '.version = $v')
  ENCODED=$(echo "$UPDATED_CONTENT" | base64 -w 0)

  COMMIT_RESP=$(GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
    gh api repos/supersuit-tech/permission-slip/contents/<package>/package.json \
    -X PUT \
    -f message="chore: bump <package> version to <version>" \
    -f content="$ENCODED" \
    -f sha="$FILE_SHA")

  # Update SHA to point to the new commit so the tag lands on it
  SHA=$(echo "$COMMIT_RESP" | jq -r '.commit.sha')
  if [ -z "$SHA" ] || [ "$SHA" = "null" ]; then
    echo "::error::Failed to bump package.json — could not get new commit SHA"
    exit 1
  fi
  echo "Committed package.json bump: $SHA"
fi
```

If the file doesn't exist (404), skip this step silently and proceed with the original `SHA`.

### 5. Create and Push the Tag

Create the tag reference on GitHub directly via the API — no local branch switching or `git push` needed:

```bash
GH_HOST=github.com GH_REPO=supersuit-tech/permission-slip \
  gh api repos/supersuit-tech/permission-slip/git/refs \
  -f ref="refs/tags/<package>/v<version>" \
  -f sha="$SHA"
```

If the command fails, print the error output and stop.

### 6. Confirm

Print a summary:
```
Tagged and pushed: <package>/v<version> -> <SHA>
```
