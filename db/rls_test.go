package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
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

// TestAppBackendHasPoliciesOnAllRLSTables verifies that every RLS-enabled table
// has an allow-all policy for the app_backend role. Without this, queries from
// the Go backend silently return zero rows — a subtle bug that's hard to debug.
func TestAppBackendHasPoliciesOnAllRLSTables(t *testing.T) {
	t.Parallel()
	pool := testhelper.SetupTestDB(t)

	// Find all RLS-enabled tables that lack an app_backend policy.
	// A TO PUBLIC policy (stored as '{=}' in pg_policies.roles) also covers
	// app_backend, so we accept that as well.
	rows, err := pool.Query(context.Background(), `
		SELECT t.tablename
		FROM pg_tables t
		WHERE t.schemaname = 'public'
		  AND t.tablename != 'goose_db_version'
		  AND t.rowsecurity = true
		  AND NOT EXISTS (
		    SELECT 1 FROM pg_policies p
		    WHERE p.schemaname = 'public'
		      AND p.tablename = t.tablename
		      AND ('app_backend' = ANY(p.roles) OR p.roles = '{=}')
		  )
		ORDER BY t.tablename
	`)
	if err != nil {
		t.Fatalf("failed to query for missing policies: %v", err)
	}
	defer rows.Close()

	var checked int
	for rows.Next() {
		var tablename string
		if err := rows.Scan(&tablename); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		checked++
		t.Errorf("table %q has RLS enabled but no policy for app_backend — the Go backend will get zero rows", tablename)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error: %v", err)
	}

	// Guard against vacuous pass when migrations didn't run.
	if checked == 0 {
		// Verify there actually are RLS-enabled tables — if there aren't,
		// something is wrong with the test setup.
		var rlsCount int
		err := pool.QueryRow(context.Background(), `
			SELECT count(*) FROM pg_tables
			WHERE schemaname = 'public'
			  AND tablename != 'goose_db_version'
			  AND rowsecurity = true
		`).Scan(&rlsCount)
		if err != nil {
			t.Fatalf("failed to count RLS tables: %v", err)
		}
		if rlsCount == 0 {
			t.Fatal("no RLS-enabled tables found — migrations may not have run")
		}
	}
}
