# Supabase Connector

Supabase integration via the PostgREST REST API. Supports reading, inserting, updating, and deleting rows with PostgREST filter syntax, table allowlists, and RLS-aware API key scoping. No SQL driver or third-party SDK needed.

## Credentials

| Key | Type | Description |
|-----|------|-------------|
| `supabase_url` | URL | Project URL (e.g., `https://abc.supabase.co`). Found in Settings > API. |
| `api_key` | API Key | Supabase API key. Use `anon` key for RLS-enforced access or `service_role` key for admin access. Found in Settings > API. |

See [Supabase API docs](https://supabase.com/docs/guides/api#api-url-and-keys) for details.

## Actions

### `supabase.read` (low risk)

Read rows from a table with optional filters, column selection, ordering, and pagination.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | string | Yes | Table name to read from |
| `select` | string | No | Columns to return (PostgREST select syntax, e.g. `id,name`). Default: `*` |
| `filters` | object | No | PostgREST filter conditions as `{"column": "operator.value"}` pairs (e.g. `{"age": "gte.18", "status": "eq.active"}`) |
| `order` | string | No | Order results (e.g. `created_at.desc` or `name.asc,id.desc`) |
| `limit` | integer | No | Max rows to return (1-10000, default: 1000) |
| `offset` | integer | No | Rows to skip for pagination |
| `count_total` | boolean | No | If true, returns `total_count` via PostgREST exact count (useful for paginated UIs) |
| `allowed_tables` | string[] | No | Restrict access to these tables only |

### `supabase.insert` (medium risk)

Insert one or more rows into a table. Supports upsert via `on_conflict`.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | string | Yes | Table name to insert into |
| `rows` | object[] | Yes | 1-1000 row objects (keys are column names) |
| `returning` | string | No | Columns to return from inserted rows. Default: `*` |
| `on_conflict` | string | No | Column(s) for upsert conflict resolution (e.g. `email`) |
| `allowed_tables` | string[] | No | Restrict access to these tables only |

### `supabase.update` (high risk)

Update rows matching filters. At least one filter is required to prevent accidental full-table updates.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | string | Yes | Table name to update |
| `set` | object | Yes | Column-value pairs to set on matching rows |
| `filters` | object | Yes | PostgREST filter conditions (same format as read) |
| `returning` | string | No | Columns to return from updated rows. Default: `*` |
| `allowed_tables` | string[] | No | Restrict access to these tables only |

### `supabase.delete` (high risk)

Delete rows matching filters. At least one filter is required to prevent accidental full-table deletes.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table` | string | Yes | Table name to delete from |
| `filters` | object | Yes | PostgREST filter conditions (same format as read) |
| `returning` | string | No | Columns to return from deleted rows. Default: `*` |
| `allowed_tables` | string[] | No | Restrict access to these tables only |

## Filter Syntax

Filters use PostgREST operator syntax: `{"column": "operator.value"}`.

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `{"status": "eq.active"}` |
| `neq` | Not equals | `{"status": "neq.deleted"}` |
| `gt`, `gte` | Greater than (or equal) | `{"age": "gte.18"}` |
| `lt`, `lte` | Less than (or equal) | `{"price": "lt.100"}` |
| `like`, `ilike` | Pattern match (case-sensitive/insensitive) | `{"name": "ilike.%alice%"}` |
| `is` | IS (null, true, false) | `{"deleted_at": "is.null"}` |
| `in` | IN list | `{"role": "in.(admin,editor)"}` |
| `not.*` | Negated operator | `{"status": "not.eq.deleted"}` |

See [PostgREST operators](https://postgrest.org/en/stable/references/api/tables_views.html#operators) for the full list.

## Error Mapping

| PostgREST/PostgreSQL Error | Connector Error | Description |
|----------------------------|-----------------|-------------|
| HTTP 401 | AuthError | Invalid API key |
| HTTP 403 | AuthError | Insufficient permissions (check RLS policies) |
| `PGRST301`, `PGRST302` | AuthError | JWT expired or invalid |
| `42501` | AuthError | Insufficient privilege |
| `42P01` | ValidationError | Undefined table |
| `42703` | ValidationError | Undefined column |
| `PGRST*` (other) | ValidationError | PostgREST-specific validation errors |
| HTTP 429 | RateLimitError | Rate limit exceeded (respects `Retry-After` header) |

## Security

- **Table allowlists**: The `allowed_tables` parameter constrains which tables an agent can access. Requests for unlisted tables are rejected before any API call is made.
- **API key scoping**: The `anon` key enforces Row Level Security (RLS); the `service_role` key bypasses RLS. Choose the appropriate key for your use case.
- **Filter validation**: Column names are validated against reserved PostgREST parameters (`select`, `order`, `limit`, etc.) and checked for safe characters to prevent query parameter injection.
- **Table name validation**: Table names must contain only letters, digits, underscores, hyphens, or dots.
- **Required filters**: Update and delete actions require at least one filter to prevent accidental full-table mutations.
- **Response size cap**: Responses are limited to 10 MB to prevent runaway reads.

## File Structure

```
connectors/supabase/
  supabase.go           # Connector struct, HTTP lifecycle, manifest, credential validation
  read.go               # supabase.read action
  insert.go             # supabase.insert action (with upsert support)
  update.go             # supabase.update action
  delete.go             # supabase.delete action
  errors.go             # PostgREST error response parsing and error type mapping
  validate.go           # Table/filter/column validation helpers
  register.go           # Built-in connector registration (init)
  logo.svg              # Embedded Supabase logo
  helpers_test.go       # Shared test helpers (validCreds, validCredsWithURL)
  *_test.go             # Tests for each action
  README.md             # This file
```

## Adding a New Action

1. Create `new_action.go` with the action struct, params, and `Execute` method.
2. Add `validate()` to the params struct and use `parseAndValidate` in `Execute`.
3. Register the action in `Actions()` in `supabase.go`.
4. Add the `ManifestAction` entry in the `Manifest()` method in `supabase.go`.
5. Add tests in `new_action_test.go` using `httptest.NewServer`.
6. Update this README.
