# GitHub Connector

The GitHub connector integrates Permission Slip with the [GitHub REST API](https://docs.github.com/en/rest). It uses plain `net/http` — no third-party GitHub SDK.

## Connector ID

`github`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A GitHub personal access token (classic or fine-grained) with appropriate scopes for the actions being executed. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

## Actions

### `github.create_issue`

Creates a new issue in a GitHub repository.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `owner` | string | Yes | Repository owner (user or organization) |
| `repo` | string | Yes | Repository name |
| `title` | string | Yes | Issue title |
| `body` | string | No | Issue body (Markdown supported) |

**Response:**

```json
{
  "number": 42,
  "url": "https://api.github.com/repos/octocat/hello-world/issues/42",
  "html_url": "https://github.com/octocat/hello-world/issues/42"
}
```

**GitHub API:** `POST /repos/{owner}/{repo}/issues` ([docs](https://docs.github.com/en/rest/issues/issues#create-an-issue))

**Required token scopes:** `repo` (classic) or Issues write permission (fine-grained)

---

### `github.close_issue`

Closes an issue, optionally posting a comment before closing.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `owner` | string | Yes | — | Repository owner (user or organization) |
| `repo` | string | Yes | — | Repository name |
| `issue_number` | integer | Yes | — | Issue number to close |
| `state_reason` | string | No | `"completed"` | One of `completed` or `not_planned` |
| `comment` | string | No | — | Comment to post before closing |

**Response:**

```json
{
  "number": 10,
  "url": "https://api.github.com/repos/octocat/hello-world/issues/10",
  "html_url": "https://github.com/octocat/hello-world/issues/10",
  "state": "closed"
}
```

**GitHub API:** `PATCH /repos/{owner}/{repo}/issues/{issue_number}` ([docs](https://docs.github.com/en/rest/issues/issues#update-an-issue))

**Required token scopes:** `repo` (classic) or Issues write permission (fine-grained)

---

### `github.add_label`

Adds labels to an issue or pull request.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `owner` | string | Yes | Repository owner (user or organization) |
| `repo` | string | Yes | Repository name |
| `issue_number` | integer | Yes | Issue or pull request number |
| `labels` | array of strings | Yes | Labels to add |

**Response:** Array of label objects with `id` and `name`.

**GitHub API:** `POST /repos/{owner}/{repo}/issues/{issue_number}/labels` ([docs](https://docs.github.com/en/rest/issues/labels#add-labels-to-an-issue))

**Required token scopes:** `repo` (classic) or Issues write permission (fine-grained)

---

### `github.add_comment`

Adds a comment to an issue or pull request.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `owner` | string | Yes | Repository owner (user or organization) |
| `repo` | string | Yes | Repository name |
| `issue_number` | integer | Yes | Issue or pull request number |
| `body` | string | Yes | Comment body (Markdown supported) |

**Response:**

```json
{
  "id": 123,
  "url": "https://api.github.com/repos/octocat/hello-world/issues/comments/123",
  "html_url": "https://github.com/octocat/hello-world/issues/7#issuecomment-123",
  "body": "Looks good!"
}
```

**GitHub API:** `POST /repos/{owner}/{repo}/issues/{issue_number}/comments` ([docs](https://docs.github.com/en/rest/issues/comments#create-an-issue-comment))

**Required token scopes:** `repo` (classic) or Issues write permission (fine-grained)

---

### `github.create_pr`

Creates a pull request from a branch.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `owner` | string | Yes | — | Repository owner (user or organization) |
| `repo` | string | Yes | — | Repository name |
| `title` | string | Yes | — | Pull request title |
| `body` | string | No | — | Pull request body (Markdown supported) |
| `head` | string | Yes | — | Branch containing the changes |
| `base` | string | Yes | — | Branch to merge into |
| `draft` | boolean | No | `false` | Whether to create the PR as a draft |

**Response:**

```json
{
  "number": 99,
  "url": "https://api.github.com/repos/octocat/hello-world/pulls/99",
  "html_url": "https://github.com/octocat/hello-world/pull/99",
  "state": "open",
  "draft": false
}
```

**GitHub API:** `POST /repos/{owner}/{repo}/pulls` ([docs](https://docs.github.com/en/rest/pulls/pulls#create-a-pull-request))

**Required token scopes:** `repo` (classic) or Pull Requests write permission (fine-grained)

---

### `github.merge_pr`

Merges an open pull request.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `owner` | string | Yes | — | Repository owner (user or organization) |
| `repo` | string | Yes | — | Repository name |
| `pull_number` | integer | Yes | — | Pull request number |
| `merge_method` | string | No | `"merge"` | One of `merge`, `squash`, or `rebase` |

**Response:**

```json
{
  "sha": "abc123def456",
  "merged": true,
  "message": "Pull Request successfully merged"
}
```

**GitHub API:** `PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge` ([docs](https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request))

**Required token scopes:** `repo` (classic) or Pull Requests write permission (fine-grained)

---

### `github.add_reviewer`

Requests reviews on a pull request.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `owner` | string | Yes | Repository owner (user or organization) |
| `repo` | string | Yes | Repository name |
| `pull_number` | integer | Yes | Pull request number |
| `reviewers` | array of strings | Yes | GitHub usernames to request reviews from |

**Response:**

```json
{
  "number": 42,
  "url": "https://api.github.com/repos/octocat/hello-world/pulls/42",
  "html_url": "https://github.com/octocat/hello-world/pull/42"
}
```

**GitHub API:** `POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers` ([docs](https://docs.github.com/en/rest/pulls/review-requests#request-reviewers-for-a-pull-request))

**Required token scopes:** `repo` (classic) or Pull Requests write permission (fine-grained)

---

### `github.create_release`

Creates a tagged release in a repository.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `owner` | string | Yes | — | Repository owner (user or organization) |
| `repo` | string | Yes | — | Repository name |
| `tag_name` | string | Yes | — | The name of the tag for this release |
| `name` | string | No | — | The name of the release |
| `body` | string | No | — | Release notes (Markdown supported) |
| `draft` | boolean | No | `false` | Whether to create as a draft release |
| `prerelease` | boolean | No | `false` | Whether to mark as a pre-release |

**Response:**

```json
{
  "id": 1,
  "url": "https://api.github.com/repos/octocat/hello-world/releases/1",
  "html_url": "https://github.com/octocat/hello-world/releases/tag/v1.0.0",
  "tag_name": "v1.0.0",
  "name": "Release 1.0",
  "draft": false
}
```

**GitHub API:** `POST /repos/{owner}/{repo}/releases` ([docs](https://docs.github.com/en/rest/releases/releases#create-a-release))

**Required token scopes:** `repo` (classic) or Contents write permission (fine-grained)

---

### `github.create_branch`

Creates a new branch from an existing branch or tag.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `owner` | string | Yes | Repository owner (user or organization) |
| `repo` | string | Yes | Repository name |
| `branch_name` | string | Yes | Name for the new branch |
| `from_ref` | string | Yes | Branch or ref to create from (e.g. `"main"`, `"develop"`, or `"tags/v1.0"`) |

Bare branch names (e.g. `"main"`) are automatically expanded to `"heads/main"`. To branch from a tag, use the full ref path (e.g. `"tags/v1.0"`).

**Response:**

```json
{
  "ref": "refs/heads/feature-branch",
  "url": "https://api.github.com/repos/octocat/hello-world/git/refs/heads/feature-branch",
  "object": {
    "sha": "abc123def456"
  }
}
```

**GitHub API:** `GET /repos/{owner}/{repo}/git/ref/{ref}` + `POST /repos/{owner}/{repo}/git/refs` ([docs](https://docs.github.com/en/rest/git/refs#create-a-reference))

**Required token scopes:** `repo` (classic) or Contents write permission (fine-grained)

## Error Handling

The connector maps GitHub API responses to typed connector errors:

| GitHub Status | Connector Error | HTTP Response |
|---------------|-----------------|---------------|
| 401, 403 | `AuthError` | 502 Bad Gateway |
| 422 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| 403 + rate limit headers | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

## Adding a New Action

Each action lives in its own file. To add one:

1. Create `connectors/github/<action>.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle — it handles JSON marshaling, auth headers, response checking, and error mapping.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `github.go`.
5. Add the action to the `Manifest()` return value inside `github.go` — include a `ParametersSchema` (see below).
6. Add one or more `ManifestTemplate` entries for common permission presets.
7. Add tests in `<action>_test.go` using `httptest.NewServer` and `newForTest()`.

The `do` method means each action file only contains what's unique: parameter parsing, validation, request body shape, and response shape. All shared HTTP concerns (auth, Content-Type, error mapping) are handled once.

## Parameters Schema

Each action declares a `parameters_schema` (JSON Schema) in its manifest entry. This schema:

- **Drives the approval UI** — the frontend renders parameter descriptions, types, and enum choices automatically instead of showing raw key-value pairs
- **Documents the API contract** — agents can use the schema to validate parameters before submitting requests
- **Populates the database** — auto-seeded into `connector_actions.parameters_schema` on startup

When adding a new action, define its `ParametersSchema` as a `json.RawMessage` in the manifest. Use `connectors.TrimIndent()` to keep the inline JSON readable:

```go
{
    ActionType:  "github.close_issue",
    Name:        "Close Issue",
    Description: "Close an open issue",
    RiskLevel:   "low",
    ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
        "type": "object",
        "required": ["owner", "repo", "issue_number"],
        "properties": {
            "owner": {
                "type": "string",
                "description": "Repository owner (user or organization)"
            },
            "repo": {
                "type": "string",
                "description": "Repository name"
            },
            "issue_number": {
                "type": "integer",
                "description": "Issue number to close"
            }
        }
    }`)),
}
```

The schema supports standard JSON Schema properties: `type`, `description`, `required`, `enum`, and `default`. The frontend reads these to render rich parameter displays in the approval review modal.

## Manifest

Connector reference data (the `connectors`, `connector_actions`, and `connector_required_credentials` rows) is declared in the `Manifest()` method on `GitHubConnector`. The server auto-upserts these DB rows on startup from the manifest — no manual SQL or seed files needed.

When adding a new action, add it to the `Manifest()` return value with a `ParametersSchema`.

## File Structure

```
connectors/github/
├── github.go               # GitHubConnector struct, New(), Manifest(), do(), ValidateCredentials()
├── create_issue.go          # github.create_issue action
├── close_issue.go           # github.close_issue action
├── add_label.go             # github.add_label action
├── add_comment.go           # github.add_comment action
├── create_pr.go             # github.create_pr action
├── merge_pr.go              # github.merge_pr action
├── add_reviewer.go          # github.add_reviewer action
├── create_release.go        # github.create_release action
├── create_branch.go         # github.create_branch action
├── validation.go            # Shared validation helpers (parseAndValidate, requireOwnerRepo, etc.)
├── response.go              # Shared HTTP response → typed error mapping
├── github_test.go           # Connector-level tests
├── helpers_test.go          # Shared test helpers (validCreds)
├── create_issue_test.go     # Tests for github.create_issue
├── close_issue_test.go      # Tests for github.close_issue
├── add_label_test.go        # Tests for github.add_label
├── add_comment_test.go      # Tests for github.add_comment
├── create_pr_test.go        # Tests for github.create_pr
├── merge_pr_test.go         # Tests for github.merge_pr
├── add_reviewer_test.go     # Tests for github.add_reviewer
├── create_release_test.go   # Tests for github.create_release
├── create_branch_test.go    # Tests for github.create_branch
└── README.md                # This file
```

## Testing

All tests use `httptest.NewServer` to mock the GitHub API — no real API calls are made.

```bash
go test ./connectors/github/... -v
```
