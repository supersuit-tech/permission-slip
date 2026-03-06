package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteAction implements connectors.Action for mysql.delete.
type deleteAction struct {
	conn *MySQLConnector
}

type deleteParams struct {
	Table         string                 `json:"table"`
	Where         map[string]interface{} `json:"where"`
	AllowedTables []string               `json:"allowed_tables"`
}

func (p *deleteParams) validate() error {
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if !isValidIdentifier(p.Table) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid table name: %q", p.Table)}
	}
	if len(p.Where) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: where (unconditional deletes are not allowed)"}
	}
	if err := checkTableAllowed(p.Table, p.AllowedTables); err != nil {
		return err
	}

	for col := range p.Where {
		if !isValidIdentifier(col) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid column name in where: %q", col)}
		}
	}
	return nil
}

// Execute deletes rows from a MySQL table and returns the number of rows affected.
func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build WHERE clause with deterministic ordering.
	whereCols := make([]string, 0, len(params.Where))
	for col := range params.Where {
		whereCols = append(whereCols, col)
	}
	sort.Strings(whereCols)

	whereParts := make([]string, len(whereCols))
	var args []any
	for i, col := range whereCols {
		whereParts[i] = quoteIdentifier(col) + " = ?"
		args = append(args, params.Where[col])
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		quoteIdentifier(params.Table),
		strings.Join(whereParts, " AND "),
	)

	db, err := a.conn.openConn(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("MySQL delete timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MySQL delete failed: %v", err)}
	}

	rowsAffected, _ := result.RowsAffected()

	return connectors.JSONResult(map[string]any{
		"rows_affected": rowsAffected,
	})
}
