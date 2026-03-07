# Jira Connector

The Jira connector integrates Permission Slip with the [Jira Cloud REST API v3](https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/). It uses basic auth (email + API token) via plain `net/http` — no third-party Jira SDK.

## Connector ID

`jira`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `site` | Yes | Atlassian site subdomain (e.g., `mycompany` for `mycompany.atlassian.net`). Must be alphanumeric with hyphens only — validated to prevent SSRF. |
| `email` | Yes | Atlassian account email address used for API authentication. |
| `api_token` | Yes | Atlassian API token. See [Atlassian docs](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/) for how to generate one. |

The credential `auth_type` in the database is `basic`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

### Base URL

The connector dynamically constructs the API base URL from `site`:

```
https://{site}.atlassian.net/rest/api/3
```

The `site` value is validated against `^[a-zA-Z0-9][a-zA-Z0-9-]*$` to prevent SSRF attacks via host redirection.

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `jira.create_issue` | Create Issue | low | Create a new issue in a Jira project |
| `jira.update_issue` | Update Issue | medium | Update fields on an existing issue |
| `jira.transition_issue` | Transition Issue | medium | Move an issue to a different status (e.g., "In Progress", "Done") |
| `jira.add_comment` | Add Comment | low | Add a comment to an existing issue |
| `jira.assign_issue` | Assign Issue | low | Assign or unassign an issue |
| `jira.search` | Search Issues | low | Search for issues using JQL |

### `jira.transition_issue` — Name-Based Lookup

Transitions are specified by name (e.g., "Done") rather than numeric ID. The action:

1. Fetches available transitions for the issue via `GET /issue/{key}/transitions`
2. Matches the requested name case-insensitively
3. Returns a clear error listing available transitions if no match is found

### `jira.add_comment` and descriptions — Atlassian Document Format

The Jira v3 API requires comment and description bodies in [Atlassian Document Format (ADF)](https://developer.atlassian.com/cloud/jira/platform/apis/document/structure/). The connector automatically wraps plain text into ADF — each newline becomes a separate paragraph node.

## Error Handling

The connector maps Jira API responses to typed connector errors:

| Jira Status | Connector Error | HTTP Response |
|-------------|-----------------|---------------|
| 401 | `AuthError` | 502 Bad Gateway |
| 403 | `AuthError` | 502 Bad Gateway |
| 400 | `ValidationError` | 400 Bad Request |
| 404 | `ValidationError` | 400 Bad Request |
| 422 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

Jira error responses are parsed from their standard format (`errorMessages` array + `errors` map) and included in the error message, truncated to 512 characters.

Rate limit responses include the `Retry-After` header value when present.

## Security

- **SSRF prevention**: The `site` credential is validated with a strict regex (`^[a-zA-Z0-9][a-zA-Z0-9-]*$`) before being interpolated into the base URL. This prevents path traversal, host redirection, and port injection.
- **OOM prevention**: Response bodies are capped at 10 MB via `io.LimitReader` to guard against malicious or oversized API responses.

## Adding a New Action

1. Create `connectors/jira/<action_name>.go` with a params struct, `validate()`, and `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle.
3. Return `connectors.JSONResult(respBody)` to wrap the response.
4. Register in `Actions()` in `jira.go`.
5. Add the action to `Manifest()` in `manifest.go` with a `ParametersSchema`.
6. Add tests in `<action_name>_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/jira/
├── jira.go                  # JiraConnector struct, New(), Actions(), ValidateCredentials(), apiBase(), do()
├── manifest.go              # Manifest() with action schemas, credentials, templates
├── response.go              # Shared HTTP response → typed error mapping
├── adf.go                   # plainTextToADF() — shared ADF conversion helper
├── create_issue.go          # jira.create_issue action
├── update_issue.go          # jira.update_issue action
├── transition_issue.go      # jira.transition_issue action (name-based lookup)
├── add_comment.go           # jira.add_comment action
├── assign_issue.go          # jira.assign_issue action
├── search.go                # jira.search action (JQL)
├── *_test.go                # Tests for each action + connector + response
├── helpers_test.go          # Shared test helpers (validCreds)
└── README.md                # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Jira API — no real API calls are made.

```bash
go test ./connectors/jira/... -v
```
