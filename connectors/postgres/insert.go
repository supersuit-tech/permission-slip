package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/pkg/sqldb"
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
		cols = sqldb.SortedKeys(p.Rows[0])
	}
	if len(cols) == 0 {
		return &connectors.ValidationError{Message: "rows must contain at least one column"}
	}
	for _, col := range cols {
		if err := validateIdentifier(col, "column"); err != nil {
			return &connectors.ValidationError{Message: err.Error()}
		}
	}
	return validateReturningCols(p.Returning)
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

	connStr, err := getConnString(req)
	if err != nil {
		return nil, err
	}

	timeout := a.conn.resolveTimeout(params.TimeoutSeconds)

	env, err := prepareTx(ctx, a.conn, connStr, timeout)
	if err != nil {
		return nil, err
	}
	defer env.Close()

	// Determine columns.
	cols := params.Columns
	if len(cols) == 0 {
		cols = sqldb.SortedKeys(params.Rows[0])
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

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s%s",
		quoteIdentifier(params.Table),
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "),
		buildReturningClause(params.Returning),
	)

	return execMutation(env, query, args, len(params.Returning) > 0, "executing insert")
}
