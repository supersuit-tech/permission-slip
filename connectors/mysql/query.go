package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// queryAction implements connectors.Action for mysql.query.
// It executes parameterized SELECT queries (read-only).
type queryAction struct {
	conn *MySQLConnector
}

type queryParams struct {
	SQL           string   `json:"sql"`
	Args          []any    `json:"args"`
	AllowedTables []string `json:"allowed_tables"`
	RowLimit      int      `json:"row_limit"`
}

func (p *queryParams) validate() error {
	if p.SQL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sql"}
	}
	normalized := strings.ToUpper(strings.TrimSpace(p.SQL))
	if !strings.HasPrefix(normalized, "SELECT") {
		return &connectors.ValidationError{Message: "only SELECT queries are allowed"}
	}
	// Block dangerous keywords that could modify data.
	for _, keyword := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "TRUNCATE", "GRANT", "REVOKE"} {
		// Check for the keyword as a standalone word (preceded by space/start and followed by space/end).
		if containsKeyword(normalized, keyword) {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("query contains disallowed keyword: %s", keyword),
			}
		}
	}
	if p.RowLimit < 0 {
		return &connectors.ValidationError{Message: "row_limit must be non-negative"}
	}
	return nil
}

// containsKeyword checks if a SQL keyword appears as a standalone word in the query.
func containsKeyword(sql, keyword string) bool {
	idx := 0
	for {
		pos := strings.Index(sql[idx:], keyword)
		if pos < 0 {
			return false
		}
		absPos := idx + pos
		// Check character before: must be start-of-string or non-alpha.
		if absPos > 0 {
			prev := sql[absPos-1]
			if (prev >= 'A' && prev <= 'Z') || (prev >= 'a' && prev <= 'z') || prev == '_' {
				idx = absPos + len(keyword)
				continue
			}
		}
		// Check character after: must be end-of-string or non-alpha.
		endPos := absPos + len(keyword)
		if endPos < len(sql) {
			next := sql[endPos]
			if (next >= 'A' && next <= 'Z') || (next >= 'a' && next <= 'z') || next == '_' {
				idx = endPos
				continue
			}
		}
		return true
	}
}

// Execute runs a parameterized SELECT query and returns the result rows as JSON.
func (a *queryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params queryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Apply row limit: use param value, fall back to connector default.
	rowLimit := a.conn.rowLimit
	if params.RowLimit > 0 {
		rowLimit = params.RowLimit
	}

	// If allowed_tables is set, verify the query doesn't reference other tables.
	if len(params.AllowedTables) > 0 {
		if err := validateQueryTables(params.SQL, params.AllowedTables); err != nil {
			return nil, err
		}
	}

	// Append LIMIT if not already present. We request one extra row beyond the
	// limit so we can detect truncation and tell the caller if more rows exist.
	query := params.SQL
	userSuppliedLimit := false
	normalized := strings.ToUpper(strings.TrimSpace(query))
	if !strings.Contains(normalized, "LIMIT") {
		query = fmt.Sprintf("%s LIMIT %d", strings.TrimRight(query, "; "), rowLimit+1)
	} else {
		userSuppliedLimit = true
	}

	db, err := a.conn.openConn(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Start a read-only transaction as defense in depth — even if keyword
	// filtering is bypassed, MySQL will reject any data-modifying statements.
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("starting read-only transaction: %v", err)}
	}
	defer tx.Rollback() //nolint:errcheck // rollback of read-only tx is best-effort

	rows, err := tx.QueryContext(ctx, query, params.Args...)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("MySQL query timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MySQL query failed: %v", err)}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading column names: %v", err)}
	}

	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("scanning row: %v", err)}
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for JSON-friendly output.
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("iterating rows: %v", err)}
	}

	// Detect whether more rows existed than the limit allowed.
	truncated := false
	if !userSuppliedLimit && len(results) > rowLimit {
		results = results[:rowLimit]
		truncated = true
	}

	return connectors.JSONResult(map[string]any{
		"columns":   columns,
		"rows":      results,
		"row_count": len(results),
		"truncated": truncated,
	})
}

// validateQueryTables does a basic check that the SQL query only references
// tables in the allowlist. This is a best-effort check using FROM/JOIN parsing.
func validateQueryTables(sql string, allowedTables []string) error {
	allowed := make(map[string]bool, len(allowedTables))
	for _, t := range allowedTables {
		allowed[strings.ToLower(t)] = true
	}

	upper := strings.ToUpper(sql)
	tables := extractTableNames(upper, sql)
	for _, t := range tables {
		if !allowed[strings.ToLower(t)] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("table %q is not in the allowed tables list", t),
			}
		}
	}
	return nil
}

// extractTableNames extracts table names from FROM and JOIN clauses.
// This is intentionally conservative — it may false-positive but should not
// false-negative (miss tables).
func extractTableNames(upperSQL, originalSQL string) []string {
	var tables []string
	keywords := []string{"FROM", "JOIN"}

	for _, kw := range keywords {
		idx := 0
		for {
			pos := strings.Index(upperSQL[idx:], kw)
			if pos < 0 {
				break
			}
			absPos := idx + pos + len(kw)

			// Skip whitespace after keyword.
			for absPos < len(originalSQL) && (originalSQL[absPos] == ' ' || originalSQL[absPos] == '\t' || originalSQL[absPos] == '\n') {
				absPos++
			}

			// Read the table name (until whitespace, comma, semicolon, or paren).
			start := absPos
			for absPos < len(originalSQL) {
				ch := originalSQL[absPos]
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == ',' || ch == ';' || ch == ')' || ch == '(' {
					break
				}
				absPos++
			}
			if start < absPos {
				tableName := originalSQL[start:absPos]
				// Strip backticks if present.
				tableName = strings.Trim(tableName, "`")
				if tableName != "" {
					tables = append(tables, tableName)
				}
			}

			idx = absPos
		}
	}
	return tables
}
