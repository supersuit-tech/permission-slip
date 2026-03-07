# Linear Connector

The Linear connector integrates Permission Slip with the [Linear GraphQL API](https://developers.linear.app/docs/graphql/working-with-the-graphql-api). It uses plain `net/http` — no third-party Linear SDK.

## Connector ID

`linear`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `api_key` | Yes | A Linear personal API key. |

The credential `auth_type` in the database is `api_key`. Tokens are stored encrypted in Supabase Vault and decrypted only at execution time.

**Getting a key:** Go to [Linear Settings → API](https://linear.app/settings/api) and create a personal API key. The key is an opaque string with no fixed prefix.

**Auth header format:** Linear uses `Authorization: {api_key}` — no "Bearer" prefix, unlike most APIs.

## Actions

| Action | Risk | Description |
|--------|------|-------------|
| `linear.create_issue` | low | Create a new issue in a Linear team |
| `linear.update_issue` | low | Update fields on an existing issue |
| `linear.add_comment` | low | Add a markdown comment to an issue |
| `linear.create_project` | medium | Create a new project across one or more teams |
| `linear.search_issues` | low | Search issues with full-text search or filtered queries |

### `linear.create_issue`

Creates a new issue in a team.

**Linear API:** `mutation issueCreate`
**Required fields:** `team_id`, `title`
**Optional fields:** `description`, `assignee_id`, `priority` (0–4), `state_id`, `label_ids`, `project_id`

### `linear.update_issue`

Updates fields on an existing issue. At least one field besides `issue_id` must be provided (no-op guard).

**Linear API:** `mutation issueUpdate`
**Required fields:** `issue_id` + at least one of: `title`, `description`, `assignee_id`, `priority`, `state_id`, `label_ids`

### `linear.add_comment`

Adds a markdown comment to an issue.

**Linear API:** `mutation commentCreate`
**Required fields:** `issue_id`, `body`

### `linear.create_project`

Creates a new project associated with one or more teams.

**Linear API:** `mutation projectCreate`
**Required fields:** `team_ids`, `name`
**Optional fields:** `description`, `state` (one of: `planned`, `started`, `paused`, `completed`, `cancelled`)

### `linear.search_issues`

Searches issues using two strategies:

- **Full-text search** (no filters): Uses Linear's `issueSearch` query, which searches across titles, descriptions, and comments.
- **Filtered search** (with `team_id`, `assignee_id`, or `state`): Uses the `issues` endpoint with structured filters and title matching.

**Required fields:** `query`
**Optional filters:** `team_id`, `assignee_id`, `state`, `limit` (default 50, max 100)

**Response format:** Returns `{issues: [...], total_count: N}` — the envelope makes it easy for agents to check result counts.

## Error Handling

Linear returns errors in two ways: HTTP status codes and GraphQL `errors` array with `extensions.type`. The connector maps both:

| Linear Error | Connector Error |
|-------------|-----------------|
| `extensions.type: "authentication_error"` | `AuthError` |
| `extensions.type: "forbidden"` | `AuthError` |
| `extensions.type: "ratelimited"` | `RateLimitError` |
| `extensions.type: "validation_error"` | `ValidationError` |
| HTTP 401, 403 | `AuthError` |
| HTTP 429 | `RateLimitError` (parses `Retry-After` header) |
| Other GraphQL errors | `ExternalError` |
| Client timeout / context canceled | `TimeoutError` |

## Security

- **No injection risk:** All GraphQL queries use parameterized variables — user input never touches query strings.
- **Response body limits:** API responses are capped at 10 MB (`maxResponseBytes`) via `io.LimitReader` to prevent OOM.
- **Credential validation:** API key presence is checked both in `ValidateCredentials()` and at the start of every `doGraphQL()` call.
- **No SSRF:** The base URL is hardcoded to `https://api.linear.app/graphql` in production; only overridable via `newForTest()`.

## Adding a New Action

Each action lives in its own file. To add one (e.g., `linear.assign_issue`):

1. Create `connectors/linear/assign_issue.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.doGraphQL(ctx, creds, mutation, variables, &resp)` — it handles JSON marshaling, auth, response parsing, and error mapping.
3. Return `connectors.JSONResult(...)` to wrap the response into an `ActionResult`.
4. Register the action in `Actions()` inside `linear.go`.
5. Add the action to `Manifest()` in `manifest.go` with a `ParametersSchema`.
6. Add tests in `assign_issue_test.go` using the `graphQLHandler` from `helpers_test.go`.

Use `validatePriority()` from `linear.go` if the action accepts a priority field — it enforces the 0–4 range consistently.

## File Structure

```
connectors/linear/
├── linear.go               # LinearConnector struct, New(), Actions(), doGraphQL(), error mapping
├── manifest.go             # Manifest() — action schemas, credential requirements, templates
├── create_issue.go         # linear.create_issue
├── update_issue.go         # linear.update_issue (with no-op guard)
├── add_comment.go          # linear.add_comment
├── create_project.go       # linear.create_project (with state validation)
├── search_issues.go        # linear.search_issues (full-text + filtered)
├── linear_test.go          # Connector-level and doGraphQL tests
├── helpers_test.go         # Shared test helpers (graphQLHandler, validCreds)
├── *_test.go               # Per-action test files
└── README.md               # This file
```

## Testing

All tests use `httptest.NewServer` with the `graphQLHandler` helper to mock the Linear GraphQL API — no real API calls are made.

```bash
go test ./connectors/linear/... -v
```
