package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// insertAction implements connectors.Action for postgres.insert.
type insertAction struct {
	conn *PostgresConnector
}

type insertParams struct {
	Table          string                   `json:"table"`
	Columns        []string                 `json:"columns"`
	Rows           []map[string]interface{} `json:"rows"`
	Returning      []string                 `json:"returning"`
	TimeoutSeconds int                      `json:"timeout_seconds"`
}

func (p *insertParams) validate() error {
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if err := validateIdentifier(p.Table, "table"); err != nil {
		return &connectors.ValidationError{Message: err.Error()}
	}
	if len(p.Rows) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: rows (must contain at least one row)"}
	}
	if len(p.Rows) > 1000 {
		return &connectors.ValidationError{Message: "rows exceeds maximum of 1000 rows per insert"}
	}

	// Determine columns from explicit list or first row.
	cols := p.Columns
	if len(cols) == 0 {
		cols = sortedKeys(p.Rows[0])
	}
	if len(cols) == 0 {
		return &connectors.ValidationError{Message: "rows must contain at least one column"}
	}
	for _, col := range cols {
		if err := validateIdentifier(col, "column"); err != nil {
			return &connectors.ValidationError{Message: err.Error()}
		}
	}
	for _, col := range p.Returning {
		if col != "*" {
			if err := validateIdentifier(col, "returning column"); err != nil {
				return &connectors.ValidationError{Message: err.Error()}
			}
		}
	}
	return nil
}

// Execute inserts rows into a table and returns the count (and optionally RETURNING data).
func (a *insertAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params insertParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	connStr, ok := req.Credentials.Get("connection_string")
	if !ok || connStr == "" {
		return nil, &connectors.ValidationError{Message: "missing credential: connection_string"}
	}

	timeout := a.conn.resolveTimeout(params.TimeoutSeconds)

	db, err := a.conn.openDB(connStr, timeout)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("failed to open database: %v", err)}
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Determine columns.
	cols := params.Columns
	if len(cols) == 0 {
		cols = sortedKeys(params.Rows[0])
	}

	// Build INSERT statement.
	quotedCols := make([]string, len(cols))
	for i, col := range cols {
		quotedCols[i] = quoteIdentifier(col)
	}

	var placeholders []string
	var args []interface{}
	paramIdx := 1

	for _, row := range params.Rows {
		rowPlaceholders := make([]string, len(cols))
		for i, col := range cols {
			rowPlaceholders[i] = fmt.Sprintf("$%d", paramIdx)
			paramIdx++
			args = append(args, row[col])
		}
		placeholders = append(placeholders, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		quoteIdentifier(params.Table),
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "),
	)

	hasReturning := len(params.Returning) > 0
	if hasReturning {
		quotedReturning := make([]string, len(params.Returning))
		for i, col := range params.Returning {
			if col == "*" {
				quotedReturning[i] = "*"
			} else {
				quotedReturning[i] = quoteIdentifier(col)
			}
		}
		query += " RETURNING " + strings.Join(quotedReturning, ", ")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, mapPgError(err, "beginning transaction")
	}
	defer tx.Rollback() //nolint:errcheck

	timeoutMS := int(timeout.Milliseconds())
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", timeoutMS)); err != nil {
		return nil, mapPgError(err, "setting statement timeout")
	}

	if hasReturning {
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, mapPgError(err, "executing insert")
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading column names: %v", err)}
		}

		var returned []map[string]interface{}
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
			returned = append(returned, row)
		}
		if err := rows.Err(); err != nil {
			return nil, mapPgError(err, "iterating returned rows")
		}

		if err := tx.Commit(); err != nil {
			return nil, mapPgError(err, "committing transaction")
		}

		if returned == nil {
			returned = []map[string]interface{}{}
		}
		return connectors.JSONResult(map[string]interface{}{
			"rows_affected": len(returned),
			"returned":      returned,
		})
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, mapPgError(err, "executing insert")
	}

	if err := tx.Commit(); err != nil {
		return nil, mapPgError(err, "committing transaction")
	}

	affected, _ := result.RowsAffected()
	return connectors.JSONResult(map[string]interface{}{
		"rows_affected": affected,
	})
}
