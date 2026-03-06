package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateAction implements connectors.Action for postgres.update.
type updateAction struct {
	conn *PostgresConnector
}

type updateParams struct {
	Table          string                 `json:"table"`
	Set            map[string]interface{} `json:"set"`
	Where          map[string]interface{} `json:"where"`
	Returning      []string               `json:"returning"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
}

func (p *updateParams) validate() error {
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if err := validateIdentifier(p.Table, "table"); err != nil {
		return &connectors.ValidationError{Message: err.Error()}
	}
	if len(p.Set) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: set (must contain at least one column to update)"}
	}
	if len(p.Where) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: where (unconditional updates are not allowed)"}
	}
	for col := range p.Set {
		if err := validateIdentifier(col, "set column"); err != nil {
			return &connectors.ValidationError{Message: err.Error()}
		}
	}
	if err := validateWhereCols(p.Where); err != nil {
		return err
	}
	return validateReturningCols(p.Returning)
}

// Execute updates rows in a table and returns the affected count.
func (a *updateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateParams
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

	// Build SET clause.
	setCols := sortedKeys(params.Set)
	var setClauses []string
	var args []interface{}
	paramIdx := 1

	for _, col := range setCols {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), paramIdx))
		args = append(args, params.Set[col])
		paramIdx++
	}

	// Build WHERE clause.
	whereSQL, whereArgs := buildWhereClause(params.Where, paramIdx)
	args = append(args, whereArgs...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s%s",
		quoteIdentifier(params.Table),
		strings.Join(setClauses, ", "),
		whereSQL,
		buildReturningClause(params.Returning),
	)

	return execMutation(env, query, args, len(params.Returning) > 0, "executing update")
}
