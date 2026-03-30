package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/pkg/sqldb"
)

// queryAction implements connectors.Action for postgres.query.
// It executes parameterized SELECT queries in read-only mode.
type queryAction struct {
	conn *PostgresConnector
}

type queryParams struct {
	SQL            string        `json:"sql"`
	Params         []interface{} `json:"params"`
	MaxRows        int           `json:"max_rows"`
	TimeoutSeconds int           `json:"timeout_seconds"`
}

func (p *queryParams) validate() error {
	if p.SQL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sql"}
	}

	// Only allow SELECT statements (and WITH/CTE that lead to SELECT).
	normalized := strings.TrimSpace(strings.ToUpper(p.SQL))
	if !strings.HasPrefix(normalized, "SELECT") && !strings.HasPrefix(normalized, "WITH") {
		return &connectors.ValidationError{Message: "only SELECT queries are allowed (query must start with SELECT or WITH)"}
	}

	if p.MaxRows < 0 {
		return &connectors.ValidationError{Message: "max_rows must be positive"}
	}
	if p.MaxRows > 10000 {
		return &connectors.ValidationError{Message: "max_rows cannot exceed 10000"}
	}
	return nil
}

// Execute runs a read-only SELECT query and returns the result rows as JSON.
func (a *queryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params queryParams
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
	maxRows := a.conn.maxRows
	if params.MaxRows > 0 {
		maxRows = params.MaxRows
	}

	env, err := prepareTx(ctx, a.conn, connStr, timeout)
	if err != nil {
		return nil, err
	}
	defer env.Close()

	// Enforce read-only transaction to block writes even via CTEs.
	if _, err := env.Tx.ExecContext(env.Ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return nil, mapPgError(err, "setting read-only mode")
	}

	rows, err := env.Tx.QueryContext(env.Ctx, params.SQL, params.Params...)
	if err != nil {
		return nil, mapPgError(err, "executing query")
	}
	defer rows.Close()

	columns, results, err := sqldb.ScanRows(rows, maxRows, func(e error) error {
		return mapPgError(e, "iterating rows")
	})
	if err != nil {
		return nil, err
	}

	results, truncated := sqldb.DetectTruncation(results, maxRows)

	return connectors.JSONResult(map[string]interface{}{
		"columns":   columns,
		"rows":      results,
		"row_count": len(results),
		"truncated": truncated,
	})
}
