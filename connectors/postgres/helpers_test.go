package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validCreds returns a Credentials value with a valid connection string for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"connection_string": "postgres://localhost:5432/permission_slip_test?sslmode=disable",
	})
}

var tableCounter atomic.Int64

// setupTestTable creates a unique temporary table for each test to avoid
// race conditions when tests run in parallel. Returns the table name.
func setupTestTable(t *testing.T) string {
	t.Helper()

	// Generate a unique table name per test invocation.
	n := tableCounter.Add(1)
	// Sanitize test name to be a valid lowercase identifier.
	safe := strings.ToLower(strings.NewReplacer("/", "_", " ", "_", "-", "_", ".", "_").Replace(t.Name()))
	tableName := fmt.Sprintf("ct_%s_%d", safe, n)
	// Truncate to fit PostgreSQL's 63-char identifier limit.
	if len(tableName) > 63 {
		tableName = tableName[:63]
	}

	connStr := "postgres://localhost:5432/permission_slip_test?sslmode=disable"
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Skipf("skipping: cannot connect to test database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("skipping: cannot ping test database: %v", err)
	}

	_, err = db.Exec(fmt.Sprintf(`
		DROP TABLE IF EXISTS %s;
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			value INTEGER,
			active BOOLEAN DEFAULT true
		);
		INSERT INTO %s (name, value, active) VALUES
			('alpha', 10, true),
			('beta', 20, false),
			('gamma', 30, true);
	`, tableName, tableName, tableName))
	if err != nil {
		t.Fatalf("setting up test table: %v", err)
	}

	t.Cleanup(func() {
		db2, err := sql.Open("pgx", connStr)
		if err != nil {
			return
		}
		defer db2.Close()
		db2.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)) //nolint:errcheck
	})

	return tableName
}
