package snowflake

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/snowflakedb/gosnowflake"
	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/pkg/sqldb"
)

type queryAction struct {
	conn *SnowflakeConnector
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

func (a *queryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params queryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	connStr, err := getDSN(req)
	if err != nil {
		return nil, err
	}

	timeout := a.conn.resolveTimeout(params.TimeoutSeconds)
	maxRows := a.conn.maxRows
	if params.MaxRows > 0 {
		maxRows = params.MaxRows
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	db, err := sql.Open("snowflake", connStr)
	if err != nil {
		return nil, mapSnowflakeError(err, "opening connection")
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(timeout + 5*time.Second)

	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, mapSnowflakeError(err, "connecting")
	}

	sec := int(timeout.Seconds())
	if sec < 1 {
		sec = 1
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER SESSION SET STATEMENT_TIMEOUT_IN_SECONDS = %d", sec)); err != nil {
		return nil, mapSnowflakeError(err, "setting statement timeout")
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, mapSnowflakeError(err, "starting read-only transaction")
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryContext(ctx, params.SQL, params.Params...)
	if err != nil {
		return nil, mapSnowflakeError(err, "executing query")
	}
	defer rows.Close()

	columns, results, err := sqldb.ScanRows(rows, maxRows, func(e error) error {
		return mapSnowflakeError(e, "iterating rows")
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

func getDSN(req connectors.ActionRequest) (string, error) {
	connStr, ok := req.Credentials.Get("connection_string")
	if !ok {
		connStr = ""
	}
	pkPEM, _ := req.Credentials.Get("private_key_pem")
	return composeDSN(connStr, pkPEM)
}
