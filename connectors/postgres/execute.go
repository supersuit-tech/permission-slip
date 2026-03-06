package postgres

// execute.go contains shared transaction lifecycle helpers used by all actions.
// It eliminates boilerplate by providing:
//   - prepareTx: open DB → begin transaction → set statement_timeout
//   - execMutation: run a write query with optional RETURNING, commit, return result
//   - getConnString: extract connection_string from action credentials
//   - buildReturningClause: build a quoted RETURNING SQL fragment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// txEnv bundles a prepared transaction with its context and cancel function.
// Every action creates one via prepareTx, uses it for queries, and defers cleanup.
type txEnv struct {
	Tx     *sql.Tx
	Ctx    context.Context
	Cancel context.CancelFunc
	DB     *sql.DB
}

// Close cleans up the transaction environment. Call via defer after prepareTx.
func (e *txEnv) Close() {
	e.Tx.Rollback() //nolint:errcheck — cleanup only
	e.Cancel()
	e.DB.Close()
}

// prepareTx opens a database connection, begins a transaction, and sets the
// statement timeout. This eliminates the repeated boilerplate across all actions.
func prepareTx(ctx context.Context, c *PostgresConnector, connStr string, timeout time.Duration) (*txEnv, error) {
	db, err := c.openDB(connStr, timeout)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("failed to open database: %v", err)}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		db.Close()
		return nil, mapPgError(err, "beginning transaction")
	}

	timeoutMS := int(timeout.Milliseconds())
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", timeoutMS)); err != nil {
		tx.Rollback() //nolint:errcheck
		cancel()
		db.Close()
		return nil, mapPgError(err, "setting statement timeout")
	}

	return &txEnv{Tx: tx, Ctx: ctx, Cancel: cancel, DB: db}, nil
}

// getConnString extracts and validates the connection string from credentials.
func getConnString(req connectors.ActionRequest) (string, error) {
	connStr, ok := req.Credentials.Get("connection_string")
	if !ok || connStr == "" {
		return "", &connectors.ValidationError{Message: "missing credential: connection_string"}
	}
	return connStr, nil
}

// buildReturningClause builds a RETURNING clause from a list of column names.
// Returns an empty string if cols is empty.
func buildReturningClause(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	quoted := make([]string, len(cols))
	for i, col := range cols {
		if col == "*" {
			quoted[i] = "*"
		} else {
			quoted[i] = quoteIdentifier(col)
		}
	}
	return " RETURNING " + strings.Join(quoted, ", ")
}

// execMutation runs a write query (INSERT/UPDATE/DELETE) within a transaction,
// handling both the simple case (rows_affected) and the RETURNING case.
// The action parameter is used for error messages (e.g., "executing insert").
func execMutation(env *txEnv, query string, args []interface{}, hasReturning bool, action string) (*connectors.ActionResult, error) {
	if hasReturning {
		rows, err := env.Tx.QueryContext(env.Ctx, query, args...)
		if err != nil {
			return nil, mapPgError(err, action)
		}
		defer rows.Close()

		returned, err := scanRows(rows)
		if err != nil {
			return nil, err
		}

		if err := env.Tx.Commit(); err != nil {
			return nil, mapPgError(err, "committing transaction")
		}

		return connectors.JSONResult(map[string]interface{}{
			"rows_affected": len(returned),
			"returned":      returned,
		})
	}

	result, err := env.Tx.ExecContext(env.Ctx, query, args...)
	if err != nil {
		return nil, mapPgError(err, action)
	}

	if err := env.Tx.Commit(); err != nil {
		return nil, mapPgError(err, "committing transaction")
	}

	affected, _ := result.RowsAffected()
	return connectors.JSONResult(map[string]interface{}{
		"rows_affected": affected,
	})
}
