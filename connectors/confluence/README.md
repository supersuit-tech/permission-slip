# Confluence Connector

The Confluence connector integrates Permission Slip with the [Confluence Cloud REST API v2](https://developer.atlassian.com/cloud/confluence/rest/v2/intro/). It uses basic auth (email + API token) via plain `net/http` — no third-party SDK.

## Connector ID

`confluence`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `site` | Yes | Atlassian site subdomain (e.g., `mycompany` for `mycompany.atlassian.net`). Must be alphanumeric with hyphens only — validated to prevent SSRF. |
| `email` | Yes | Atlassian account email address used for API authentication. |
| `api_token` | Yes | Atlassian API token. See [Atlassian docs](https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/) for how to generate one. |

**Shared credentials:** Uses the `jira` credential service, shared with the Jira connector. Users who already have Jira credentials configured can reuse them for Confluence.

### Base URL

The connector dynamically constructs the API base URL from `site`:

```
https://{site}.atlassian.net/wiki/api/v2
```

The `site` value is validated against `^[a-zA-Z0-9][a-zA-Z0-9-]*$` to prevent SSRF attacks via host redirection.

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `confluence.create_page` | Create Page | low | Create a new page in a Confluence space |
| `confluence.update_page` | Update Page | medium | Update page title or content (requires version number for optimistic locking) |
| `confluence.get_page` | Get Page | low | Get page content and metadata (includes version number needed for updates) |
| `confluence.search` | Search | low | Search across pages using CQL (Confluence Query Language) |
| `confluence.add_comment` | Add Comment | low | Add a footer comment to a page |

### `confluence.update_page` — Optimistic Locking

Confluence uses optimistic locking to prevent concurrent edits. The `version_number` parameter must be the current version + 1. Agents should always call `get_page` first to get the current version number.

**Recommended workflow:**
1. `confluence.get_page` → note `version.number` from response
2. `confluence.update_page` → set `version_number` to current + 1

### Page Body Format

Confluence accepts page bodies in **storage format** (XHTML-like markup). The connector uses storage format by default. When reading pages via `get_page`, you can choose between `storage`, `atlas_doc_format` (ADF), or `view` (rendered HTML).

## Error Handling

The connector maps Confluence API responses to typed connector errors:

| Confluence Status | Connector Error | HTTP Response |
|-------------|-----------------|---------------|
| 401 | `AuthError` | 502 Bad Gateway |
| 403 | `AuthError` | 502 Bad Gateway |
| 400 | `ValidationError` | 400 Bad Request |
| 404 | `ValidationError` | 400 Bad Request |
| 422 | `ValidationError` | 400 Bad Request |
| 429 | `RateLimitError` | 429 Too Many Requests |
| Other 4xx/5xx | `ExternalError` | 502 Bad Gateway |
| Client timeout / context deadline | `TimeoutError` | 504 Gateway Timeout |

Confluence error responses are parsed from their standard format (`{"message": "..."}` or `{"errors": [{"title": "..."}]}`) and included in the error message.

Rate limit responses include the `Retry-After` header value when present.

## Security

- **SSRF prevention**: The `site` credential is validated with a strict regex (`^[a-zA-Z0-9][a-zA-Z0-9-]*$`) before being interpolated into the base URL. This prevents path traversal, host redirection, and port injection.
- **OOM prevention**: Response bodies are capped at 10 MB via `io.LimitReader` to guard against malicious or oversized API responses.

## Adding a New Action

1. Create `connectors/confluence/<action_name>.go` with a params struct, `validate()`, and `Execute` method.
2. Use `a.conn.do(ctx, creds, method, path, reqBody, &respBody)` for the HTTP lifecycle.
3. Return `connectors.JSONResult(respBody)` to wrap the response.
4. Register in `Actions()` in `confluence.go`.
5. Add the action to `Manifest()` in `manifest.go` with a `ParametersSchema`.
6. Add tests in `<action_name>_test.go` using `httptest.NewServer` and `newForTest()`.

## File Structure

```
connectors/confluence/
├── confluence.go            # ConfluenceConnector struct, New(), Actions(), ValidateCredentials(), apiBase(), do()
├── manifest.go              # Manifest() with action schemas, credentials, templates
├── response.go              # Shared HTTP response → typed error mapping
├── types.go                 # Shared pageResponse struct and toResult() helper
├── create_page.go           # confluence.create_page action
├── update_page.go           # confluence.update_page action (optimistic locking)
├── get_page.go              # confluence.get_page action
├── search.go                # confluence.search action (CQL)
├── add_comment.go           # confluence.add_comment action
├── *_test.go                # Tests for each action + connector + response
├── helpers_test.go          # Shared test helpers (validCreds)
└── README.md                # This file
```

## Testing

All tests use `httptest.NewServer` to mock the Confluence API — no real API calls are made.

```bash
go test ./connectors/confluence/... -v
```
