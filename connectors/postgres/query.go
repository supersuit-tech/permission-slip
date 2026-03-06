package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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

	columns, err := rows.Columns()
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading column names: %v", err)}
	}

	// Read up to maxRows+1 to detect truncation.
	var results []map[string]interface{}
	for rows.Next() {
		if len(results) >= maxRows+1 {
			break
		}

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
			val := values[i]
			// Convert []byte to string for JSON serialization.
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err, "iterating rows")
	}

	// If we read more than maxRows, truncate and flag it.
	truncated := len(results) > maxRows
	if truncated {
		results = results[:maxRows]
	}
	if results == nil {
		results = []map[string]interface{}{}
	}

	return connectors.JSONResult(map[string]interface{}{
		"columns":   columns,
		"rows":      results,
		"row_count": len(results),
		"truncated": truncated,
	})
}
