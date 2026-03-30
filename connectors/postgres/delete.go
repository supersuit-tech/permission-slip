package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteAction implements connectors.Action for postgres.delete.
type deleteAction struct {
	conn *PostgresConnector
}

type deleteParams struct {
	Table          string                 `json:"table"`
	Where          map[string]interface{} `json:"where"`
	Returning      []string               `json:"returning"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
}

func (p *deleteParams) validate() error {
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if err := validateIdentifier(p.Table, "table"); err != nil {
		return &connectors.ValidationError{Message: err.Error()}
	}
	if len(p.Where) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: where (unconditional deletes are not allowed)"}
	}
	if err := validateWhereCols(p.Where); err != nil {
		return err
	}
	return validateReturningCols(p.Returning)
}

// Execute deletes rows from a table and returns the affected count.
func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
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

	whereSQL, args := buildWhereClause(params.Where, 1)

	query := fmt.Sprintf("DELETE FROM %s WHERE %s%s",
		quoteIdentifier(params.Table),
		whereSQL,
		buildReturningClause(params.Returning),
	)

	return execMutation(env, query, args, len(params.Returning) > 0, "executing delete")
}
