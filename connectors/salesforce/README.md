# Salesforce Connector

The Salesforce connector integrates Permission Slip with the [Salesforce REST API](https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/). It uses OAuth 2.0 access tokens provided by the platform — no OAuth code in the connector itself.

## Connector ID

`salesforce`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `access_token` | Yes | OAuth 2.0 access token — injected by the platform after the user completes the OAuth flow |
| `instance_url` | Yes | Salesforce org URL (e.g., `https://myorg.salesforce.com`) — extracted from the OAuth token response |

The credential `auth_type` is `oauth2` with provider `salesforce`. The platform handles the entire OAuth lifecycle (authorize, callback, token exchange, storage, refresh). The connector receives a valid `access_token` and `instance_url` at execution time.

### Authentication

All API requests use Bearer token authentication: `Authorization: Bearer {access_token}`.

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `salesforce.create_record` | Create Record | low | Create a new record of any sObject type |
| `salesforce.update_record` | Update Record | medium | Update fields on an existing record |
| `salesforce.query` | Query (SOQL) | low | Execute a SOQL query to retrieve records |
| `salesforce.create_task` | Create Task | low | Create a task (activity) linked to a record |
| `salesforce.add_note` | Add Note | low | Add a ContentNote to a record |

### Record ID Validation

Record IDs (`record_id`, `parent_id`, `what_id`, `who_id`) are validated to be 15 or 18 alphanumeric characters, matching the Salesforce ID format. Invalid IDs return a `ValidationError` before hitting the API.

### SOQL Queries

SOQL queries are passed as-is to the Salesforce API. The `max_records` parameter (default 200, max 2000) truncates the result set client-side. When truncation occurs, the response includes `"truncated": true`.

### Task Defaults

- `status` defaults to `"Not Started"`
- `priority` defaults to `"Normal"`
- `due_date` must be in `YYYY-MM-DD` format

### ContentNote Creation (add_note)

This is a two-step operation: create a `ContentNote`, then link it to the parent record via `ContentDocumentLink`. If the note is created but linking fails, the action returns a partial success (`success: false`) with a warning message rather than an error.

## Error Mapping

Salesforce returns errors as `[{"errorCode": "...", "message": "..."}]`. These are mapped to typed connector errors:

| Salesforce Error Code | Connector Error Type |
|---|---|
| `INVALID_SESSION_ID` | `AuthError` |
| `REQUEST_LIMIT_EXCEEDED` | `RateLimitError` |
| `MALFORMED_QUERY`, `INVALID_FIELD`, `INVALID_TYPE`, `REQUIRED_FIELD_MISSING`, `STRING_TOO_LONG`, `DUPLICATE_VALUE`, `INVALID_CROSS_REFERENCE_KEY`, `FIELD_CUSTOM_VALIDATION_EXCEPTION`, `INVALID_OR_NULL_FOR_RESTRICTED_PICKLIST` | `ValidationError` |

HTTP status codes are also mapped: 401/403 → `AuthError`, 429 → `RateLimitError`, 5xx → `ExternalError`.

## API Details

- **Base URL:** `{instance_url}/services/data/v62.0/` (dynamic per org)
- **API version:** v62.0 (pinned)
- **Response limit:** 10 MB max response body
- **Request timeout:** 30 seconds

## File Structure

```
connectors/salesforce/
├── salesforce.go           # Connector struct, doJSON helper, error mapping
├── salesforce_test.go      # Interface, manifest, checkResponse tests
├── manifest.go             # ManifestProvider with schemas and templates
├── helpers_test.go         # Shared test credentials
├── create_record.go        # salesforce.create_record action
├── create_record_test.go
├── update_record.go        # salesforce.update_record action
├── update_record_test.go
├── query.go                # salesforce.query action
├── query_test.go
├── create_task.go          # salesforce.create_task action
├── create_task_test.go
├── add_note.go             # salesforce.add_note action
├── add_note_test.go
└── README.md
```

## Running Tests

```bash
go test ./connectors/salesforce/... -v
```

All tests use `httptest.Server` — no real Salesforce credentials or network access needed.
