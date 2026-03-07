# Airtable Connector

Airtable integration for structured data and no-code databases. Supports listing, creating, updating, searching, and deleting records across Airtable bases.

## Credentials

| Key | Type | Description |
|-----|------|-------------|
| `api_token` | Personal Access Token | Must start with `pat`. Create one at [airtable.com/create/tokens](https://airtable.com/create/tokens). |

## Actions

### `airtable.list_bases` (low risk)

List all bases accessible to the authenticated user.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `offset` | string | No | Pagination offset from a previous response |

### `airtable.list_records` (low risk)

List records from a table with optional filtering, sorting, and pagination.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_id` | string | Yes | Base ID (starts with `app`) |
| `table` | string | Yes | Table name or ID |
| `fields` | string[] | No | Only return these columns |
| `filter_by_formula` | string | No | Airtable formula filter |
| `max_records` | integer | No | Max records to return |
| `page_size` | integer | No | Records per page (1-100) |
| `sort` | object[] | No | Sort order (`field`, `direction`) |
| `view` | string | No | View name or ID |
| `offset` | string | No | Pagination offset |

### `airtable.get_record` (low risk)

Get a single record by ID.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_id` | string | Yes | Base ID (starts with `app`) |
| `table` | string | Yes | Table name or ID |
| `record_id` | string | Yes | Record ID (starts with `rec`) |

### `airtable.create_records` (medium risk)

Create 1-10 records in a table.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_id` | string | Yes | Base ID (starts with `app`) |
| `table` | string | Yes | Table name or ID |
| `records` | object[] | Yes | 1-10 records, each with `fields` map |

### `airtable.update_records` (medium risk)

Partial update (PATCH) 1-10 records. Only specified fields are changed.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_id` | string | Yes | Base ID (starts with `app`) |
| `table` | string | Yes | Table name or ID |
| `records` | object[] | Yes | 1-10 records, each with `id` and `fields` |

### `airtable.delete_records` (high risk)

Permanently delete 1-10 records. This cannot be undone.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_id` | string | Yes | Base ID (starts with `app`) |
| `table` | string | Yes | Table name or ID |
| `record_ids` | string[] | Yes | 1-10 record IDs (each starts with `rec`) |

### `airtable.search_records` (low risk)

Search records using an Airtable formula. Convenience wrapper around `list_records` with a required formula and default limit of 100.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `base_id` | string | Yes | Base ID (starts with `app`) |
| `table` | string | Yes | Table name or ID |
| `formula` | string | Yes | Airtable filter formula |
| `fields` | string[] | No | Only return these columns |
| `max_records` | integer | No | Max records (default: 100) |
| `sort` | object[] | No | Sort order |

## Error Mapping

| Airtable Error Type | Connector Error | Description |
|---------------------|-----------------|-------------|
| `AUTHENTICATION_REQUIRED` | AuthError | Token is invalid or expired |
| `INVALID_PERMISSIONS_OR_MODEL_NOT_FOUND` | AuthError | Token lacks access or resource doesn't exist |
| `NOT_FOUND`, `TABLE_NOT_FOUND`, `ROW_DOES_NOT_EXIST` | ExternalError (404) | Resource not found |
| `VIEW_NOT_FOUND` | ExternalError (404) | View doesn't exist |
| `INVALID_FILTER_BY_FORMULA` | ValidationError | Formula syntax error |
| `FIELD_NOT_FOUND`, `UNKNOWN_FIELD_NAME` | ValidationError | Column doesn't exist |
| `INVALID_VALUE_FOR_COLUMN` | ValidationError | Wrong value type for field |
| `CANNOT_UPDATE_COMPUTED_FIELD` | ValidationError | Can't write to formula/lookup fields |
| HTTP 429 | RateLimitError | 5 requests/second per base exceeded |

## Rate Limits

Airtable enforces a strict rate limit of **5 requests per second per base**. The connector returns a `RateLimitError` with `RetryAfter` when this is hit. Batch operations (up to 10 records per request) help stay within limits.

## File Structure

```
connectors/airtable/
  airtable.go             # Connector struct, HTTP lifecycle, error mapping
  manifest.go             # Manifest (actions, credentials, templates)
  list_bases.go           # airtable.list_bases
  list_records.go         # airtable.list_records + shared record types
  get_record.go           # airtable.get_record
  create_records.go       # airtable.create_records
  update_records.go       # airtable.update_records
  delete_records.go       # airtable.delete_records
  search_records.go       # airtable.search_records
  helpers_test.go         # Shared test helpers (validCreds)
  *_test.go               # Tests for each action
  README.md               # This file
```

## Adding a New Action

1. Create `new_action.go` with the action struct, params, and `Execute` method.
2. Add `validate()` to the params struct and use `parseAndValidate` in `Execute`.
3. Register the action in `Actions()` in `airtable.go`.
4. Add the `ManifestAction` entry in `manifest.go`.
5. Add tests in `new_action_test.go` using `httptest.NewServer`.
6. Update this README.
