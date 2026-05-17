package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestDatabaseConnectivity(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	var result int
	err := tx.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}

func TestMigrationsApplied(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	var n int
	err := tx.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM sqlite_master
		 WHERE type = 'table'
		   AND name IN ('users', 'profiles', 'subscriptions', 'audit_events')`,
	).Scan(&n)
	if err != nil {
		t.Fatalf("failed to verify core tables: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected consolidated SQLite schema to define 4 core tables, found %d", n)
	}
}
