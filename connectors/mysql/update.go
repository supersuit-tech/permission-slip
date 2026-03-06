package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateAction implements connectors.Action for mysql.update.
type updateAction struct {
	conn *MySQLConnector
}

type updateParams struct {
	Table          string                 `json:"table"`
	Set            map[string]interface{} `json:"set"`
	Where          map[string]interface{} `json:"where"`
	AllowedTables  []string               `json:"allowed_tables"`
	AllowedColumns []string               `json:"allowed_columns"`
}

func (p *updateParams) validate() error {
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if !isValidIdentifier(p.Table) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid table name: %q", p.Table)}
	}
	if len(p.Set) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: set"}
	}
	if len(p.Where) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: where (unconditional updates are not allowed)"}
	}
	if err := checkTableAllowed(p.Table, p.AllowedTables); err != nil {
		return err
	}

	// Validate identifiers and collect columns for allowlist check.
	var allCols []string
	for col := range p.Set {
		if !isValidIdentifier(col) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid column name in set: %q", col)}
		}
		allCols = append(allCols, col)
	}
	for col := range p.Where {
		if !isValidIdentifier(col) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid column name in where: %q", col)}
		}
		allCols = append(allCols, col)
	}
	return checkColumnsAllowed(allCols, p.AllowedColumns)
}

// Execute updates rows in a MySQL table and returns the number of rows affected.
func (a *updateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build SET clause with deterministic ordering.
	setCols := make([]string, 0, len(params.Set))
	for col := range params.Set {
		setCols = append(setCols, col)
	}
	sort.Strings(setCols)

	setParts := make([]string, len(setCols))
	var args []any
	for i, col := range setCols {
		setParts[i] = quoteIdentifier(col) + " = ?"
		args = append(args, params.Set[col])
	}

	// Build WHERE clause with deterministic ordering.
	whereCols := make([]string, 0, len(params.Where))
	for col := range params.Where {
		whereCols = append(whereCols, col)
	}
	sort.Strings(whereCols)

	whereParts := make([]string, len(whereCols))
	for i, col := range whereCols {
		whereParts[i] = quoteIdentifier(col) + " = ?"
		args = append(args, params.Where[col])
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		quoteIdentifier(params.Table),
		strings.Join(setParts, ", "),
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
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("MySQL update timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MySQL update failed: %v", err)}
	}

	rowsAffected, _ := result.RowsAffected()

	return connectors.JSONResult(map[string]any{
		"rows_affected": rowsAffected,
	})
}
