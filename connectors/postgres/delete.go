package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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

// Execute deletes rows from a table and returns the affected count.
func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
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

	// Build WHERE clause.
	whereCols := sortedKeys(params.Where)
	var whereClauses []string
	var args []interface{}
	paramIdx := 1

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

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		quoteIdentifier(params.Table),
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
			return nil, mapPgError(err, "executing delete")
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
		return nil, mapPgError(err, "executing delete")
	}

	if err := tx.Commit(); err != nil {
		return nil, mapPgError(err, "committing transaction")
	}

	affected, _ := result.RowsAffected()
	return connectors.JSONResult(map[string]interface{}{
		"rows_affected": affected,
	})
}
