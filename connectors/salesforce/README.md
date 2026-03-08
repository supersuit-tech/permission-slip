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

### Instance URL Validation

The `instance_url` is validated at multiple points to prevent SSRF attacks:

1. **At storage time:** `extractTokenExtraData` in the OAuth callback validates that instance_url is a well-formed HTTPS URL before persisting it.
2. **At credential validation time:** `ValidateCredentials` verifies the URL uses HTTPS and points to a `*.salesforce.com` or `*.force.com` domain.
3. **At execution time:** `apiBaseURL` re-validates the URL before constructing API endpoints.

The `access_token` credential is always sourced from the vault and set after merging extra_data, ensuring that extra_data fields cannot overwrite security-critical credentials.

## Actions

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `salesforce.create_record` | Create Record | low | Create a new record of any sObject type |
| `salesforce.update_record` | Update Record | medium | Update fields on an existing record |
| `salesforce.query` | Query (SOQL) | low | Execute a SOQL query to retrieve records |
| `salesforce.create_task` | Create Task | low | Create a task (activity) linked to a record |
| `salesforce.add_note` | Add Note | low | Add a ContentNote to a record |
| `salesforce.create_opportunity` | Create Opportunity | low | Create a new Opportunity with typed fields (name, stage, close date, amount, account) |
| `salesforce.update_opportunity` | Update Opportunity | medium | Update Opportunity stage, amount, close date, name, or description |
| `salesforce.create_lead` | Create Lead | low | Create a new Lead with typed fields (name, company, email, phone, etc.) |
| `salesforce.convert_lead` | Convert Lead | **high** | Convert a Lead into an Account, Contact, and optionally an Opportunity — **irreversible** |
| `salesforce.delete_record` | Delete Record | **high** | Permanently delete any record by sObject type and ID |
| `salesforce.describe_object` | Describe Object | low | Retrieve full schema and field metadata for any sObject type |
| `salesforce.list_reports` | List Reports | low | List all available Salesforce reports |
| `salesforce.run_report` | Run Report | low | Execute a report and return results (summary or detail) |

### Record ID Validation

Record IDs (`record_id`, `lead_id`, `account_id`, `contact_id`, etc.) are validated to be 15 or 18 alphanumeric characters, matching the Salesforce ID format. Invalid IDs return a `ValidationError` before hitting the API.

### Date Validation

Fields like `close_date` are validated to be in `YYYY-MM-DD` format **and** calendar-valid (e.g. `2026-02-30` is rejected). This prevents cryptic Salesforce API errors when dates are malformed.

### SOQL Queries

SOQL queries are passed as-is to the Salesforce API. The `max_records` parameter (default 200, max 2000) truncates the result set client-side. When truncation occurs, the response includes `"truncated": true`.

### Task Defaults

- `status` defaults to `"Not Started"`
- `priority` defaults to `"Normal"`
- `due_date` must be in `YYYY-MM-DD` format

### ContentNote Creation (add_note)

This is a two-step operation: create a `ContentNote`, then link it to the parent record via `ContentDocumentLink`. If the note is created but linking fails, the action returns a partial success (`success: false`) with a warning message rather than an error.

### Opportunity and Lead Actions

Named actions like `create_opportunity` and `create_lead` use typed, validated parameter schemas. They delegate to the same REST endpoints as `create_record` but provide:
- **Type-safe required fields** with clear error messages
- **Input validation** (date format, non-negative amounts, email format)
- **`record_url`** in the response for direct browser navigation

**Amounts:** The `amount` field uses a pointer (`*float64`) so that an explicit `0` is sent to Salesforce rather than omitted. Omitting the field and sending `0` have different semantics in some Salesforce workflows.

### Lead Conversion

`convert_lead` uses the `POST /sobjects/Lead/{id}/convert` endpoint. It:
- Always creates an Account and Contact (unless existing IDs are provided)
- Creates an Opportunity by default (pass `do_not_create_opportunity: true` to skip)
- Returns `record_url` links for every created record
- Is **irreversible** — once converted, a lead cannot be unconverted via the API

### Schema Discovery

`describe_object` returns the full Salesforce object metadata including all field names, types, picklist values, and relationship information. The response is passed through as-is from the Salesforce API (it can be large for objects with many fields).

### Reports

`list_reports` returns an empty array (`[]`) when no reports exist, never `null`. `run_report` accepts an optional `include_details` flag — when `false` (the default), only summary data is returned; when `true`, row-level detail is included.

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
├── salesforce.go              # Connector struct, doJSON helper, error mapping, shared validators
├── salesforce_test.go         # Interface, manifest, checkResponse tests
├── manifest.go                # ManifestProvider with schemas and templates for all actions
├── helpers_test.go            # Shared test credentials
├── register.go                # Registers connector in the global registry
├── create_record.go           # salesforce.create_record (generic)
├── create_record_test.go
├── update_record.go           # salesforce.update_record (generic)
├── update_record_test.go
├── query.go                   # salesforce.query (SOQL)
├── query_test.go
├── create_task.go             # salesforce.create_task
├── create_task_test.go
├── add_note.go                # salesforce.add_note (ContentNote)
├── add_note_test.go
├── create_opportunity.go      # salesforce.create_opportunity (typed)
├── create_opportunity_test.go
├── update_opportunity.go      # salesforce.update_opportunity (typed)
├── update_opportunity_test.go
├── create_lead.go             # salesforce.create_lead (typed)
├── create_lead_test.go
├── convert_lead.go            # salesforce.convert_lead (irreversible)
├── convert_lead_test.go
├── delete_record.go           # salesforce.delete_record (high risk)
├── delete_record_test.go
├── describe_object.go         # salesforce.describe_object (schema discovery)
├── describe_object_test.go
├── list_reports.go            # salesforce.list_reports
├── list_reports_test.go
├── run_report.go              # salesforce.run_report
├── run_report_test.go
└── README.md
```

## Running Tests

```bash
go test ./connectors/salesforce/... -v
```

All tests use `httptest.Server` — no real Salesforce credentials or network access needed.
