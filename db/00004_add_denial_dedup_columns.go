package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

// Migration 00004 reconciles the denial-dedup schema onto pre-existing
// databases.
//
// Background: denial_reason, action_fingerprint, and the
// idx_approvals_denial_dedup index were originally introduced by editing the
// squashed 00001_init.sql in place (PR #1286). goose only ever runs 00001
// once, so any database created before that edit never received them. Every
// query that selects the full approvals column set (e.g.
// ListApprovalsByApproverPaginated) then fails with
// "no such column: denial_reason", which surfaces to clients as a 500
// "Failed to list approvals" — breaking the pending-approvals list on both web
// and mobile.
//
// Fresh databases already get these objects from 00001_init.sql, so this
// migration is written to be idempotent: it only adds what is missing. That
// keeps it safe on both upgraded installs (columns absent) and fresh installs
// (columns already present), since both kinds of database exist in the wild.
//
// This is a Go migration rather than SQL because SQLite has no
// "ADD COLUMN IF NOT EXISTS" and a single unconditional ALTER would crash
// whichever set of databases doesn't match it.
func init() {
	goose.AddMigrationContext(upAddDenialDedupColumns, downAddDenialDedupColumns)
}

func upAddDenialDedupColumns(ctx context.Context, tx *sql.Tx) error {
	if err := addApprovalsColumnIfMissing(ctx, tx, "denial_reason",
		`ALTER TABLE approvals ADD COLUMN denial_reason TEXT `+
			`CHECK (denial_reason IS NULL OR length(denial_reason) <= 500)`); err != nil {
		return err
	}
	if err := addApprovalsColumnIfMissing(ctx, tx, "action_fingerprint",
		`ALTER TABLE approvals ADD COLUMN action_fingerprint TEXT `+
			`CHECK (action_fingerprint IS NULL OR length(action_fingerprint) <= 64)`); err != nil {
		return err
	}
	// Partial index used by the denial-dedup lookup (db/approval_dedup.go).
	// action_fingerprint is guaranteed to exist by this point.
	if _, err := tx.ExecContext(ctx,
		`CREATE INDEX IF NOT EXISTS idx_approvals_denial_dedup `+
			`ON approvals(agent_id, approver_id, action_fingerprint, denied_at DESC) `+
			`WHERE status = 'denied'`); err != nil {
		return fmt.Errorf("create idx_approvals_denial_dedup: %w", err)
	}
	return nil
}

func downAddDenialDedupColumns(ctx context.Context, tx *sql.Tx) error {
	// Only the index is dropped on the way down. The columns are nullable and
	// harmless to leave in place, and SQLite cannot DROP a column referenced by
	// a CHECK constraint without a full table rebuild. Down migrations are a
	// dev-only convenience here, so this stays simple and non-destructive.
	if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_approvals_denial_dedup`); err != nil {
		return fmt.Errorf("drop idx_approvals_denial_dedup: %w", err)
	}
	return nil
}

// addApprovalsColumnIfMissing runs alterSQL only when the named column is absent
// from the approvals table, making column additions idempotent across fresh and
// upgraded databases.
func addApprovalsColumnIfMissing(ctx context.Context, tx *sql.Tx, column, alterSQL string) error {
	exists, err := approvalsColumnExists(ctx, tx, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := tx.ExecContext(ctx, alterSQL); err != nil {
		return fmt.Errorf("add approvals column %s: %w", column, err)
	}
	return nil
}

// approvalsColumnExists reports whether the approvals table already has a column
// with the given name.
func approvalsColumnExists(ctx context.Context, tx *sql.Tx, column string) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info('approvals')`)
	if err != nil {
		return false, fmt.Errorf("inspect approvals columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
