# PostgreSQL Connector

The PostgreSQL connector integrates Permission Slip with PostgreSQL databases. It uses Go's `database/sql` with the [`pgx`](https://github.com/jackc/pgx) driver — no ORM or query builder.

## Connector ID

`postgres`

## Credentials

| Key | Required | Description |
|-----|----------|-------------|
| `connection_string` | Yes | A PostgreSQL connection string in URI format: `postgres://user:password@host:port/dbname?sslmode=require` |

The credential `auth_type` in the database is `custom`. Connection strings are stored encrypted in Supabase Vault and decrypted only at execution time.

**Connection string format:**
```
postgres://username:password@hostname:5432/database_name?sslmode=require
```

Common `sslmode` values: `disable`, `require`, `verify-ca`, `verify-full`. Use `require` or stricter for production.

## Actions

### `postgres.query`

Execute a parameterized read-only SELECT query. The transaction is set to READ ONLY, preventing any writes even via CTEs.

**Risk level:** low

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `sql` | string | Yes | — | Parameterized SELECT query (use `$1`, `$2`, ... for placeholders) |
| `params` | array | No | `[]` | Positional parameters for the query placeholders |
| `max_rows` | integer | No | `1000` | Maximum rows to return (1–10,000) |
| `timeout_seconds` | integer | No | `30` | Statement timeout in seconds (1–300) |

**Response:**

```json
{
  "columns": ["id", "name", "email"],
  "rows": [
    {"id": 1, "name": "Alice", "email": "alice@example.com"},
    {"id": 2, "name": "Bob", "email": "bob@example.com"}
  ],
  "row_count": 2,
  "truncated": false
}
```

When `truncated` is `true`, more rows matched than `max_rows` — the result is capped.

---

### `postgres.insert`

Insert one or more rows into a table using parameterized values.

**Risk level:** medium

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `table` | string | Yes | Target table name (`schema.table` format supported) |
| `rows` | array of objects | Yes | Row objects to insert (keys are column names) |
| `columns` | array of strings | No | Explicit column list. If omitted, keys from the first row are used. |
| `returning` | array of strings | No | Columns to return via `RETURNING` clause (use `["*"]` for all) |
| `timeout_seconds` | integer | No | Statement timeout in seconds (default: 30) |

**Response:**

```json
{
  "rows_affected": 2,
  "returned": [
    {"id": 10, "name": "Alice"},
    {"id": 11, "name": "Bob"}
  ]
}
```

The `returned` field is only present when `returning` is specified.

---

### `postgres.update`

Update rows matching a WHERE clause. A WHERE clause is always required — unconditional updates are rejected.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `table` | string | Yes | Target table name |
| `set` | object | Yes | Column-value pairs to update |
| `where` | object | Yes | Column-value pairs for WHERE (ANDed together). Use `null` values for `IS NULL`. |
| `returning` | array of strings | No | Columns to return via `RETURNING` clause |
| `timeout_seconds` | integer | No | Statement timeout in seconds (default: 30) |

**Response:**

```json
{
  "rows_affected": 1,
  "returned": [
    {"id": 5, "name": "Alice", "active": false}
  ]
}
```

---

### `postgres.delete`

Delete rows matching a WHERE clause. A WHERE clause is always required — unconditional deletes are rejected.

**Risk level:** high

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `table` | string | Yes | Target table name |
| `where` | object | Yes | Column-value pairs for WHERE (ANDed together). Use `null` values for `IS NULL`. |
| `returning` | array of strings | No | Columns to return via `RETURNING` clause |
| `timeout_seconds` | integer | No | Statement timeout in seconds (default: 30) |

**Response:**

```json
{
  "rows_affected": 3
}
```

## Security Model

| Protection | How it works |
|-----------|--------------|
| **No SQL interpolation** | All user-supplied values go through parameterized placeholders (`$1`, `$2`, ...) |
| **Identifier validation** | Table and column names are validated against `^[a-zA-Z_][a-zA-Z0-9_]*$` and double-quoted in generated SQL |
| **Read-only queries** | `postgres.query` uses `SET TRANSACTION READ ONLY` — even writable CTEs are blocked |
| **Required WHERE** | `postgres.update` and `postgres.delete` reject empty WHERE clauses |
| **Statement timeout** | Configurable per-action (default 30s, max 5min) — prevents runaway queries |
| **Row limits** | `postgres.query` caps results (default 1000, max 10000) with truncation detection |
| **Isolated connections** | Each action execution opens and closes its own `sql.DB` with `MaxOpenConns(1)` |
| **Credentials at execution time** | Connection strings are decrypted from vault only when an action runs |

## Error Handling

The connector maps PostgreSQL error codes to typed connector errors:

| PostgreSQL Error | Connector Error | HTTP Response |
|------------------|-----------------|---------------|
| SQLSTATE 28000/28P01 (auth failure) | `AuthError` | 502 Bad Gateway |
| SQLSTATE 42501 (permission denied) | `AuthError` | 502 Bad Gateway |
| SQLSTATE 42601 (syntax error) | `ValidationError` | 400 Bad Request |
| SQLSTATE 42P01/42703 (object not found) | `ValidationError` | 400 Bad Request |
| SQLSTATE 23xxx (constraint violation) | `ValidationError` | 400 Bad Request |
| SQLSTATE 57014 (statement timeout) | `TimeoutError` | 504 Gateway Timeout |
| Connection refused / unreachable | `ExternalError` | 502 Bad Gateway |
| Context deadline exceeded | `TimeoutError` | 504 Gateway Timeout |

## Adding a New Action

Each action lives in its own file. To add one (e.g., `postgres.upsert`):

1. Create `connectors/postgres/upsert.go` with a params struct, `validate()`, and an `Execute` method.
2. Use `a.conn.openDB()` to get a `*sql.DB`, build your SQL with `quoteIdentifier()` for safety, and use parameterized placeholders.
3. Return `connectors.JSONResult(result)` to wrap the response.
4. Register the action in `Actions()` inside `postgres.go`.
5. Add the action to the `Manifest()` return value with a `ParametersSchema`.
6. Add tests using the real test database (see Testing below).

## File Structure

```
connectors/postgres/
├── postgres.go           # PostgresConnector struct, New(), Manifest(), Actions(), ValidateCredentials()
├── query.go              # postgres.query action (read-only SELECT)
├── insert.go             # postgres.insert action
├── update.go             # postgres.update action
├── delete.go             # postgres.delete action
├── sanitize.go           # Identifier validation and quoting
├── errors.go             # PostgreSQL error → typed connector error mapping, scanRows()
├── postgres_test.go      # Connector-level tests (ID, manifest, credentials)
├── query_test.go         # Query action tests
├── insert_test.go        # Insert action tests
├── update_test.go        # Update action tests
├── delete_test.go        # Delete action tests
├── sanitize_test.go      # Identifier validation tests
├── helpers_test.go       # Shared test helpers (validCreds, setupTestTable)
└── README.md             # This file
```

## Testing

Tests run against a real PostgreSQL database (`permission_slip_test`). Each test creates a unique temporary table to allow parallel execution without conflicts.

```bash
go test ./connectors/postgres/... -v
```

Tests that require a database connection will skip automatically if the test database is unavailable.
