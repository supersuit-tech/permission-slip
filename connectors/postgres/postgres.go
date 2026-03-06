// Package postgres implements the PostgreSQL connector for the Permission Slip
// connector execution layer. It uses database/sql with the pgx driver to execute
// parameterized queries against user-provided PostgreSQL databases.
//
// Security model:
//   - All queries use parameterized placeholders — no string interpolation
//   - Statement timeouts prevent runaway queries
//   - Row limits cap SELECT result sizes
//   - Connection strings are decrypted only at execution time
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for database/sql

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultStatementTimeout = 30 * time.Second
	defaultMaxRows          = 1000
)

// PostgresConnector owns the shared configuration used by all PostgreSQL actions.
type PostgresConnector struct {
	statementTimeout time.Duration
	maxRows          int
}

// New creates a PostgresConnector with sensible defaults.
func New() *PostgresConnector {
	return &PostgresConnector{
		statementTimeout: defaultStatementTimeout,
		maxRows:          defaultMaxRows,
	}
}

// ID returns "postgres", matching the connectors.id in the database.
func (c *PostgresConnector) ID() string { return "postgres" }

// Manifest returns the connector's metadata manifest.
func (c *PostgresConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "postgres",
		Name:        "PostgreSQL",
		Description: "Read and write PostgreSQL databases with parameterized queries, row limits, and statement timeouts",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "postgres.query",
				Name:        "Run SQL Query",
				Description: "Execute a read-only SELECT query with parameterized values. Results are capped at max_rows.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sql"],
					"properties": {
						"sql": {
							"type": "string",
							"description": "Parameterized SELECT query. Use $1, $2, ... for placeholders. Example: SELECT * FROM users WHERE email = $1"
						},
						"params": {
							"type": "array",
							"items": {},
							"description": "Positional parameters for the query placeholders"
						},
						"max_rows": {
							"type": "integer",
							"minimum": 1,
							"maximum": 10000,
							"description": "Maximum number of rows to return (default: 1000)"
						},
						"timeout_seconds": {
							"type": "integer",
							"minimum": 1,
							"maximum": 300,
							"description": "Statement timeout in seconds (default: 30)"
						}
					}
				}`)),
			},
			{
				ActionType:  "postgres.insert",
				Name:        "Insert Rows",
				Description: "Insert one or more rows into a table. Supports bulk inserts and RETURNING clause.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "rows"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Target table name (schema.table format supported)"
						},
						"columns": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Column names to insert into. If omitted, keys from the first row are used."
						},
						"rows": {
							"type": "array",
							"items": {
								"type": "object"
							},
							"minItems": 1,
							"description": "Array of row objects to insert (keys are column names)"
						},
						"returning": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Columns to return from inserted rows (RETURNING clause)"
						},
						"timeout_seconds": {
							"type": "integer",
							"minimum": 1,
							"maximum": 300,
							"description": "Statement timeout in seconds (default: 30)"
						}
					}
				}`)),
			},
			{
				ActionType:  "postgres.update",
				Name:        "Update Rows",
				Description: "Update rows matching a WHERE clause. Unconditional updates are not allowed.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "set", "where"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Target table name (schema.table format supported)"
						},
						"set": {
							"type": "object",
							"description": "Column-value pairs to update"
						},
						"where": {
							"type": "object",
							"description": "Column-value pairs for the WHERE clause (ANDed together)"
						},
						"returning": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Columns to return from updated rows (RETURNING clause)"
						},
						"timeout_seconds": {
							"type": "integer",
							"minimum": 1,
							"maximum": 300,
							"description": "Statement timeout in seconds (default: 30)"
						}
					}
				}`)),
			},
			{
				ActionType:  "postgres.delete",
				Name:        "Delete Rows",
				Description: "Delete rows matching a WHERE clause. Unconditional deletes are not allowed.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "where"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Target table name (schema.table format supported)"
						},
						"where": {
							"type": "object",
							"description": "Column-value pairs for the WHERE clause (ANDed together). At least one condition is required."
						},
						"returning": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Columns to return from deleted rows (RETURNING clause)"
						},
						"timeout_seconds": {
							"type": "integer",
							"minimum": 1,
							"maximum": 300,
							"description": "Statement timeout in seconds (default: 30)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "postgres",
				AuthType:        "custom",
				InstructionsURL: "https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_postgres_query_readonly",
				ActionType:  "postgres.query",
				Name:        "Run read-only queries",
				Description: "Agent can run any SELECT query. SQL and params are agent-controlled.",
				Parameters:  json.RawMessage(`{"sql":"*","params":"*"}`),
			},
			{
				ID:          "tpl_postgres_insert",
				ActionType:  "postgres.insert",
				Name:        "Insert rows",
				Description: "Agent can insert rows into any table.",
				Parameters:  json.RawMessage(`{"table":"*","columns":"*","rows":"*"}`),
			},
			{
				ID:          "tpl_postgres_update",
				ActionType:  "postgres.update",
				Name:        "Update rows",
				Description: "Agent can update rows in any table with WHERE constraints.",
				Parameters:  json.RawMessage(`{"table":"*","set":"*","where":"*"}`),
			},
			{
				ID:          "tpl_postgres_delete",
				ActionType:  "postgres.delete",
				Name:        "Delete rows",
				Description: "Agent can delete rows from any table with WHERE constraints.",
				Parameters:  json.RawMessage(`{"table":"*","where":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *PostgresConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"postgres.query":  &queryAction{conn: c},
		"postgres.insert": &insertAction{conn: c},
		"postgres.update": &updateAction{conn: c},
		"postgres.delete": &deleteAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// valid connection_string.
func (c *PostgresConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	connStr, ok := creds.Get("connection_string")
	if !ok || connStr == "" {
		return &connectors.ValidationError{Message: "missing required credential: connection_string"}
	}

	// Validate the connection string is a parseable URL.
	u, err := url.Parse(connStr)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid connection_string: %v", err)}
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return &connectors.ValidationError{Message: "connection_string must use postgres:// or postgresql:// scheme"}
	}
	return nil
}

// openDB opens a short-lived database connection using the provided connection
// string and verifies connectivity with a Ping. The caller must close the
// returned *sql.DB.
func (c *PostgresConnector) openDB(connStr string, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	// Limit to a single connection — each action execution is isolated.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(timeout + 5*time.Second)

	// Ping to fail fast with a clear error if the database is unreachable,
	// rather than surfacing cryptic errors on the first query.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	return db, nil
}

// resolveTimeout returns the effective statement timeout from the action
// parameters, capped by the connector's default.
func (c *PostgresConnector) resolveTimeout(paramTimeout int) time.Duration {
	if paramTimeout > 0 {
		t := time.Duration(paramTimeout) * time.Second
		if t > 5*time.Minute {
			t = 5 * time.Minute
		}
		return t
	}
	return c.statementTimeout
}
