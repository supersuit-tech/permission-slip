// Package mysql implements the MySQL connector for the Permission Slip
// connector execution layer. It uses database/sql with the go-sql-driver/mysql
// driver to execute parameterized queries against MySQL databases.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// defaultTimeout is the statement timeout applied to every query.
	defaultTimeout = 30 * time.Second

	// defaultRowLimit caps SELECT result sizes to avoid dumping entire tables.
	defaultRowLimit = 1000

	// maxInsertRows caps the number of rows per INSERT to prevent massive
	// single-statement operations that could lock tables or time out.
	maxInsertRows = 1000

	// credKeyDSN is the credential key for the MySQL connection string.
	credKeyDSN = "dsn"
)

// MySQLConnector owns shared configuration used by all MySQL actions.
type MySQLConnector struct {
	timeout  time.Duration
	rowLimit int

	// openDB is the function used to open a database connection. It defaults
	// to sql.Open but can be overridden in tests.
	openDB func(dsn string) (*sql.DB, error)
}

// New creates a MySQLConnector with sensible defaults.
func New() *MySQLConnector {
	return &MySQLConnector{
		timeout:  defaultTimeout,
		rowLimit: defaultRowLimit,
		openDB:   defaultOpenDB,
	}
}

func defaultOpenDB(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

// ID returns "mysql", matching the connectors.id in the database.
func (c *MySQLConnector) ID() string { return "mysql" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *MySQLConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "mysql",
		Name:        "MySQL",
		Description: "MySQL database connector with parameterized queries, table/column allowlists, and read-only enforcement for SELECT operations",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "mysql.query",
				Name:        "Query",
				Description: "Execute a parameterized SELECT query in a read-only transaction. Supports table allowlists and automatic row limiting. Results include a truncated flag when the row limit is reached.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sql"],
					"properties": {
						"sql": {
							"type": "string",
							"description": "Parameterized SELECT query (use ? placeholders)"
						},
						"args": {
							"type": "array",
							"items": {},
							"description": "Query parameter values (positional, matching ? placeholders)"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Table allowlist — query is rejected if it references other tables"
						},
						"row_limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 10000,
							"description": "Maximum rows to return (default 1000)"
						}
					}
				}`)),
			},
			{
				ActionType:  "mysql.insert",
				Name:        "Insert",
				Description: "Insert rows into a table using parameterized queries. Supports table and column allowlists. Maximum 1000 rows per request.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "rows"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Target table name"
						},
						"rows": {
							"type": "array",
							"items": {
								"type": "object"
							},
							"minItems": 1,
							"description": "Array of row objects (column name -> value)"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Table allowlist — insert is rejected if table is not listed"
						},
						"allowed_columns": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Column allowlist — insert is rejected if any column is not listed"
						}
					}
				}`)),
			},
			{
				ActionType:  "mysql.update",
				Name:        "Update",
				Description: "Update rows using parameterized queries with required WHERE clauses. Supports table and column allowlists. Unconditional updates are rejected.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "set", "where"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Target table name"
						},
						"set": {
							"type": "object",
							"description": "Column-value pairs to update"
						},
						"where": {
							"type": "object",
							"description": "WHERE conditions as column-value equality pairs (AND-joined)"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Table allowlist — update is rejected if table is not listed"
						},
						"allowed_columns": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Column allowlist — update is rejected if any column is not listed"
						}
					}
				}`)),
			},
			{
				ActionType:  "mysql.delete",
				Name:        "Delete",
				Description: "Delete rows using parameterized queries with required WHERE clauses. Supports table allowlists. Unconditional deletes are rejected.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["table", "where"],
					"properties": {
						"table": {
							"type": "string",
							"description": "Target table name"
						},
						"where": {
							"type": "object",
							"description": "WHERE conditions as column-value equality pairs (AND-joined)"
						},
						"allowed_tables": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Table allowlist — delete is rejected if table is not listed"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "mysql", AuthType: "custom", InstructionsURL: "https://github.com/go-sql-driver/mysql#dsn-data-source-name"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_mysql_query_any",
				ActionType:  "mysql.query",
				Name:        "Query any table",
				Description: "Agent can run SELECT queries against any table.",
				Parameters:  json.RawMessage(`{"sql":"*","args":"*"}`),
			},
			{
				ID:          "tpl_mysql_query_restricted",
				ActionType:  "mysql.query",
				Name:        "Query specific tables",
				Description: "Agent can only query allowed tables with a row limit.",
				Parameters:  json.RawMessage(`{"sql":"*","args":"*","allowed_tables":["users","orders"],"row_limit":100}`),
			},
			{
				ID:          "tpl_mysql_insert",
				ActionType:  "mysql.insert",
				Name:        "Insert into specific tables",
				Description: "Agent can insert rows into allowed tables and columns.",
				Parameters:  json.RawMessage(`{"table":"*","rows":"*","allowed_tables":["orders"],"allowed_columns":["customer_id","product","quantity"]}`),
			},
			{
				ID:          "tpl_mysql_update",
				ActionType:  "mysql.update",
				Name:        "Update specific tables",
				Description: "Agent can update rows in allowed tables and columns.",
				Parameters:  json.RawMessage(`{"table":"*","set":"*","where":"*","allowed_tables":["orders"],"allowed_columns":["status","updated_at"]}`),
			},
			{
				ID:          "tpl_mysql_delete",
				ActionType:  "mysql.delete",
				Name:        "Delete from specific tables",
				Description: "Agent can delete rows from allowed tables.",
				Parameters:  json.RawMessage(`{"table":"*","where":"*","allowed_tables":["temp_records"]}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *MySQLConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"mysql.query":  &queryAction{conn: c},
		"mysql.insert": &insertAction{conn: c},
		"mysql.update": &updateAction{conn: c},
		"mysql.delete": &deleteAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty DSN for connecting to MySQL.
func (c *MySQLConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	dsn, ok := creds.Get(credKeyDSN)
	if !ok || dsn == "" {
		return &connectors.ValidationError{Message: "missing required credential: dsn"}
	}
	return nil
}

// openConn opens a MySQL connection using the DSN from credentials, sets the
// statement timeout, and returns it. The caller must close the connection.
func (c *MySQLConnector) openConn(ctx context.Context, creds connectors.Credentials) (*sql.DB, error) {
	dsn, ok := creds.Get(credKeyDSN)
	if !ok || dsn == "" {
		return nil, &connectors.ValidationError{Message: "dsn credential is missing or empty"}
	}

	db, err := c.openDB(dsn)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("opening MySQL connection: %v", err)}
	}

	// Limit to a single connection since we open/close per request.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Verify connectivity.
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("MySQL connection timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MySQL connection failed: %v", err)}
	}

	// Set statement timeout to prevent runaway queries.
	timeoutSecs := int(c.timeout.Seconds())
	if _, err := db.ExecContext(ctx, fmt.Sprintf("SET SESSION max_execution_time = %d", timeoutSecs*1000)); err != nil {
		db.Close()
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("setting statement timeout: %v", err)}
	}

	return db, nil
}

// isValidIdentifier checks that a SQL identifier (table or column name) contains
// only safe characters. This is a defense-in-depth check; parameterized queries
// handle values, but identifiers cannot be parameterized in MySQL.
func isValidIdentifier(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	// Must not start with a digit.
	if name[0] >= '0' && name[0] <= '9' {
		return false
	}
	return true
}

// quoteIdentifier wraps a validated identifier in backticks for MySQL.
// Caller must validate with isValidIdentifier first.
func quoteIdentifier(name string) string {
	return "`" + name + "`"
}

// checkTableAllowed validates that a table name is in the allowlist (if provided).
func checkTableAllowed(table string, allowedTables []string) error {
	if len(allowedTables) == 0 {
		return nil
	}
	for _, t := range allowedTables {
		if strings.EqualFold(t, table) {
			return nil
		}
	}
	return &connectors.ValidationError{
		Message: fmt.Sprintf("table %q is not in the allowed tables list", table),
	}
}

// checkColumnsAllowed validates that all columns are in the allowlist (if provided).
func checkColumnsAllowed(columns []string, allowedColumns []string) error {
	if len(allowedColumns) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(allowedColumns))
	for _, c := range allowedColumns {
		allowed[strings.ToLower(c)] = true
	}
	for _, c := range columns {
		if !allowed[strings.ToLower(c)] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("column %q is not in the allowed columns list", c),
			}
		}
	}
	return nil
}
