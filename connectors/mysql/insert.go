package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// insertAction implements connectors.Action for mysql.insert.
type insertAction struct {
	conn *MySQLConnector
}

type insertParams struct {
	Table          string                   `json:"table"`
	Rows           []map[string]interface{} `json:"rows"`
	AllowedTables  []string                 `json:"allowed_tables"`
	AllowedColumns []string                 `json:"allowed_columns"`
}

func (p *insertParams) validate() error {
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if !isValidIdentifier(p.Table) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid table name: %q", p.Table)}
	}
	if len(p.Rows) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: rows"}
	}
	if len(p.Rows) > maxInsertRows {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("too many rows: %d exceeds maximum of %d per insert", len(p.Rows), maxInsertRows),
		}
	}
	if err := checkTableAllowed(p.Table, p.AllowedTables); err != nil {
		return err
	}

	// Collect all column names across all rows.
	colSet := make(map[string]bool)
	for _, row := range p.Rows {
		if len(row) == 0 {
			return &connectors.ValidationError{Message: "rows must not contain empty objects"}
		}
		for col := range row {
			if !isValidIdentifier(col) {
				return &connectors.ValidationError{Message: fmt.Sprintf("invalid column name: %q", col)}
			}
			colSet[col] = true
		}
	}

	cols := make([]string, 0, len(colSet))
	for col := range colSet {
		cols = append(cols, col)
	}
	return checkColumnsAllowed(cols, p.AllowedColumns)
}

// Execute inserts rows into a MySQL table and returns the number of rows affected.
func (a *insertAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params insertParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Collect all unique column names and sort for deterministic ordering.
	colSet := make(map[string]bool)
	for _, row := range params.Rows {
		for col := range row {
			colSet[col] = true
		}
	}
	columns := make([]string, 0, len(colSet))
	for col := range colSet {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	// Build the INSERT statement.
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = quoteIdentifier(col)
	}

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = "?"
	}
	rowPlaceholder := "(" + strings.Join(placeholders, ", ") + ")"

	rowPlaceholders := make([]string, len(params.Rows))
	var args []any
	for i, row := range params.Rows {
		rowPlaceholders[i] = rowPlaceholder
		for _, col := range columns {
			args = append(args, row[col])
		}
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		quoteIdentifier(params.Table),
		strings.Join(quotedCols, ", "),
		strings.Join(rowPlaceholders, ", "),
	)

	db, err := a.conn.openConn(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: "MySQL insert timed out"}
		}
		return nil, &connectors.ExternalError{Message: "MySQL insert failed"}
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return connectors.JSONResult(map[string]any{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
	})
}
