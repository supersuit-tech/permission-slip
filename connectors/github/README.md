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

Each action lives in its own file. To add one (e.g., `github.close_issue`):

1. Create `connectors/github/close_issue.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle — it handles JSON marshaling, auth headers, response checking, and error mapping.
3. Return `connectors.JSONResult(respBody)` to wrap the response struct into an `ActionResult`.
4. Register the action in `Actions()` inside `github.go`.
5. Add the action to the `Manifest()` return value inside `github.go` — include a `ParametersSchema` (see below).
6. Add tests in `close_issue_test.go` using `httptest.NewServer` and `newForTest()`.

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
├── github.go             # GitHubConnector struct, New(), Manifest(), do(), ValidateCredentials()
├── create_issue.go       # github.create_issue action
├── merge_pr.go           # github.merge_pr action
├── response.go           # Shared HTTP response → typed error mapping
├── github_test.go        # Connector-level tests
├── helpers_test.go       # Shared test helpers (validCreds)
├── create_issue_test.go  # Create issue action tests
├── merge_pr_test.go      # Merge PR action tests
└── README.md             # This file
```

## Testing

All tests use `httptest.NewServer` to mock the GitHub API — no real API calls are made.

```bash
go test ./connectors/github/... -v
```
