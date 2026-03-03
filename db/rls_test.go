package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// TestRLSEnabledOnAllTables verifies that every application table in the public
// schema has Row-Level Security enabled. This prevents future migrations from
// accidentally creating tables without RLS, which would expose them via the
// Supabase PostgREST data API.
func TestRLSEnabledOnAllTables(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)

	rows, err := pool.Query(context.Background(), `
		SELECT tablename, rowsecurity
		FROM pg_tables
		WHERE schemaname = 'public'
		  AND tablename != 'goose_db_version'
		ORDER BY tablename
	`)
	if err != nil {
		t.Fatalf("failed to query pg_tables: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var tablename string
		var rowsecurity bool
		if err := rows.Scan(&tablename, &rowsecurity); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		count++
		if !rowsecurity {
			t.Errorf("table %q does not have RLS enabled", tablename)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error: %v", err)
	}
	if count == 0 {
		t.Fatal("no tables found in public schema — migrations may not have run")
	}
}
