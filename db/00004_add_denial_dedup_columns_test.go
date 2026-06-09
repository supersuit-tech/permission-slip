package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// TestMigration00004ReconcilesDenialDedupColumns simulates the exact state of a
// database created before PR #1286 (which added denial_reason /
// action_fingerprint by editing the squashed 00001_init.sql in place, so goose
// never re-applied them). It verifies that running the 00004 up migration adds
// the missing columns, that the full approvals projection then loads without a
// "no such column" error — the failure behind the 500 "Failed to list
// approvals" — and that re-running the migration is a no-op on a database that
// already has the columns.
func TestMigration00004ReconcilesDenialDedupColumns(t *testing.T) {
	ctx := context.Background()
	conn := openPreDenialDedupApprovalsDB(t)

	// Precondition: the columns are absent, reproducing the broken install.
	for _, col := range []string{"denial_reason", "action_fingerprint"} {
		if approvalsColumnExistsDB(t, conn, col) {
			t.Fatalf("precondition failed: column %q should be absent before migration", col)
		}
	}

	runUp00004(t, conn)

	for _, col := range []string{"denial_reason", "action_fingerprint"} {
		if !approvalsColumnExistsDB(t, conn, col) {
			t.Errorf("expected column %q to exist after migration", col)
		}
	}

	// The full projection used by every approvals list/get query must now load.
	rows, err := conn.QueryContext(ctx, `SELECT `+approvalColumns+` FROM approvals`)
	if err != nil {
		t.Fatalf("select full approvals projection after migration: %v", err)
	}
	defer rows.Close()
	var count int
	for rows.Next() {
		if _, err := scanApproval(rows); err != nil {
			t.Fatalf("scanApproval: %v", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 seeded approval, got %d", count)
	}

	// Idempotent: a second run (as on a fresh DB where 00001_init.sql already
	// created these objects) must not error on a duplicate column.
	runUp00004(t, conn)
}

// openPreDenialDedupApprovalsDB creates an approvals table matching the schema
// of a deployment that predates PR #1286: it has bulk_group_id (added later by
// 00003) but is missing denial_reason and action_fingerprint. A single pending
// approval is seeded so the projection scan has a row to read.
func openPreDenialDedupApprovalsDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "pre_denial.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	if _, err := conn.Exec(`
CREATE TABLE approvals (
    approval_id TEXT PRIMARY KEY,
    approver_id TEXT NOT NULL,
    action TEXT NOT NULL,
    context TEXT NOT NULL,
    status TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    approved_at TEXT,
    denied_at TEXT,
    cancelled_at TEXT,
    created_at TEXT NOT NULL,
    agent_id INTEGER NOT NULL,
    execution_status TEXT,
    execution_result TEXT,
    executed_at TEXT,
    resource_details TEXT,
    bulk_group_id TEXT
);`); err != nil {
		t.Fatalf("create pre-#1286 approvals table: %v", err)
	}

	if _, err := conn.Exec(
		`INSERT INTO approvals
		   (approval_id, approver_id, action, context, status, expires_at, created_at, agent_id)
		 VALUES ('a1', 'user-1', '{}', '{}', 'pending',
		   strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+1 day'),
		   strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), 1)`); err != nil {
		t.Fatalf("seed pending approval: %v", err)
	}
	return conn
}

func runUp00004(t *testing.T, conn *sql.DB) {
	t.Helper()
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if err := upAddDenialDedupColumns(context.Background(), tx); err != nil {
		_ = tx.Rollback()
		t.Fatalf("upAddDenialDedupColumns: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func approvalsColumnExistsDB(t *testing.T, conn *sql.DB, column string) bool {
	t.Helper()
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	exists, err := approvalsColumnExists(context.Background(), tx, column)
	if err != nil {
		t.Fatalf("approvalsColumnExists(%q): %v", column, err)
	}
	return exists
}
