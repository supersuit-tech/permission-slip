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
	for col := range p.Where {
		if err := validateIdentifier(col, "where column"); err != nil {
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

// Execute updates rows in a table and returns the affected count.
func (a *updateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateParams
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
	whereCols := sortedKeys(params.Where)
	var whereClauses []string
	for _, col := range whereCols {
		val := params.Where[col]
		if val == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL", quoteIdentifier(col)))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), paramIdx))
			args = append(args, val)
			paramIdx++
		}
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		quoteIdentifier(params.Table),
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
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
			return nil, mapPgError(err, "executing update")
		}
		defer rows.Close()

		returned, err := scanRows(rows)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, mapPgError(err, "committing transaction")
		}

		return connectors.JSONResult(map[string]interface{}{
			"rows_affected": len(returned),
			"returned":      returned,
		})
	}

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, mapPgError(err, "executing update")
	}

	if err := tx.Commit(); err != nil {
		return nil, mapPgError(err, "committing transaction")
	}

	affected, _ := result.RowsAffected()
	return connectors.JSONResult(map[string]interface{}{
		"rows_affected": affected,
	})
}
