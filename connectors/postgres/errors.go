package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// mapPgError maps PostgreSQL errors to typed connector errors.
func mapPgError(err error, action string) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Check for context/timeout errors first.
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("PostgreSQL %s timed out: %v", action, err)}
	}

	// pgx error codes are embedded in the error message.
	// SQLSTATE 28000 = invalid_authorization_specification
	// SQLSTATE 28P01 = invalid_password
	if containsAny(msg, "28000", "28P01", "password authentication failed", "no pg_hba.conf entry") {
		return &connectors.AuthError{Message: fmt.Sprintf("PostgreSQL authentication failed: %v", err)}
	}

	// Connection refused / unreachable.
	if containsAny(msg, "connection refused", "no such host", "could not connect", "connection reset") {
		return &connectors.ExternalError{Message: fmt.Sprintf("PostgreSQL connection failed: %v", err)}
	}

	// Permission denied (SQLSTATE 42501).
	if strings.Contains(msg, "42501") || strings.Contains(msg, "permission denied") {
		return &connectors.AuthError{Message: fmt.Sprintf("PostgreSQL permission denied: %v", err)}
	}

	// Syntax error or invalid SQL (SQLSTATE 42xxx).
	if strings.Contains(msg, "42601") || strings.Contains(msg, "syntax error") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL SQL syntax error: %v", err)}
	}

	// Undefined table/column (SQLSTATE 42P01, 42703).
	if containsAny(msg, "42P01", "42703", "does not exist") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL object not found: %v", err)}
	}

	// Constraint violations (SQLSTATE 23xxx).
	if strings.Contains(msg, "23") && containsAny(msg, "violates", "constraint", "duplicate key", "null value") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL constraint violation: %v", err)}
	}

	// Statement timeout (SQLSTATE 57014).
	if strings.Contains(msg, "57014") || strings.Contains(msg, "canceling statement due to statement timeout") {
		return &connectors.TimeoutError{Message: fmt.Sprintf("PostgreSQL statement timed out: %v", err)}
	}

	// Read-only transaction violation (SQLSTATE 25006).
	if strings.Contains(msg, "25006") || strings.Contains(msg, "cannot execute") && strings.Contains(msg, "read-only") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL read-only transaction violation: %v", err)}
	}

	return &connectors.ExternalError{Message: fmt.Sprintf("PostgreSQL %s failed: %v", action, err)}
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// scanRows reads all rows from a sql.Rows into a slice of maps.
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading column names: %v", err)}
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("scanning row: %v", err)}
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err, "iterating rows")
	}
	if results == nil {
		results = []map[string]interface{}{}
	}
	return results, nil
}
