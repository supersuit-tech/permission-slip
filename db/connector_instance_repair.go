package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// danglingConnectorRepairSQL removes connector instances left behind after an
// OAuth connection or static credential is deleted.
//
// SQLite does not declare a foreign key from agent_connector_credentials to
// oauth_connections or credentials, so deleting either target leaves the
// binding in place. Postgres cascades the binding away but leaves the
// agent_connectors row. Either way the instance stays selectable and shows up
// in capabilities with an empty display name.
//
// The statements are idempotent. The SQLite migration
// db/migrations_sqlite/00016_drop_dangling_connector_instances.sql must contain
// each one (TestDanglingRepairSQLMatchesSQLiteMigration). The Postgres
// migration is the boolean-typed equivalent and is not byte-identical.
var danglingConnectorRepairSQL = []string{
	// Drop bindings whose target row is gone.
	`DELETE FROM agent_connector_credentials
	WHERE (oauth_connection_id IS NOT NULL AND oauth_connection_id NOT IN (SELECT id FROM oauth_connections))
	   OR (credential_id IS NOT NULL AND credential_id NOT IN (SELECT id FROM credentials))`,

	// An unbound default with exactly one sibling that still has a binding
	// should stop being the default. Clear it first so the partial unique
	// index allows the sibling to take the flag.
	`UPDATE agent_connectors
	SET is_default = 0
	WHERE is_default = 1
	  AND NOT EXISTS (
	    SELECT 1 FROM agent_connector_credentials acc
	    WHERE acc.agent_id = agent_connectors.agent_id
	      AND acc.approver_id = agent_connectors.approver_id
	      AND acc.connector_id = agent_connectors.connector_id
	      AND acc.connector_instance_id = agent_connectors.connector_instance_id
	  )
	  AND (
	    SELECT COUNT(*) FROM agent_connectors sib
	    WHERE sib.agent_id = agent_connectors.agent_id
	      AND sib.approver_id = agent_connectors.approver_id
	      AND sib.connector_id = agent_connectors.connector_id
	      AND sib.connector_instance_id <> agent_connectors.connector_instance_id
	      AND EXISTS (
	        SELECT 1 FROM agent_connector_credentials acc
	        WHERE acc.agent_id = sib.agent_id
	          AND acc.approver_id = sib.approver_id
	          AND acc.connector_id = sib.connector_id
	          AND acc.connector_instance_id = sib.connector_instance_id
	      )
	  ) = 1`,

	// Promote that sole bound sibling when the pair no longer has a default.
	`UPDATE agent_connectors
	SET is_default = 1
	WHERE is_default = 0
	  AND EXISTS (
	    SELECT 1 FROM agent_connector_credentials acc
	    WHERE acc.agent_id = agent_connectors.agent_id
	      AND acc.approver_id = agent_connectors.approver_id
	      AND acc.connector_id = agent_connectors.connector_id
	      AND acc.connector_instance_id = agent_connectors.connector_instance_id
	  )
	  AND NOT EXISTS (
	    SELECT 1 FROM agent_connectors d
	    WHERE d.agent_id = agent_connectors.agent_id
	      AND d.approver_id = agent_connectors.approver_id
	      AND d.connector_id = agent_connectors.connector_id
	      AND d.is_default = 1
	  )
	  AND (
	    SELECT COUNT(*) FROM agent_connectors sib
	    WHERE sib.agent_id = agent_connectors.agent_id
	      AND sib.approver_id = agent_connectors.approver_id
	      AND sib.connector_id = agent_connectors.connector_id
	      AND EXISTS (
	        SELECT 1 FROM agent_connector_credentials acc
	        WHERE acc.agent_id = sib.agent_id
	          AND acc.approver_id = sib.approver_id
	          AND acc.connector_id = sib.connector_id
	          AND acc.connector_instance_id = sib.connector_instance_id
	      )
	  ) = 1`,

	// Revoke standing approvals scoped to the orphan instances we are about to drop.
	`UPDATE standing_approvals
	SET status = 'revoked', revoked_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
	WHERE status = 'active'
	  AND connector_instance_id IS NOT NULL
	  AND EXISTS (
	    SELECT 1 FROM agent_connectors ac
	    WHERE ac.agent_id = standing_approvals.agent_id
	      AND ac.approver_id = standing_approvals.user_id
	      AND ac.connector_instance_id = standing_approvals.connector_instance_id
	      AND ac.is_default = 0
	      AND NOT EXISTS (
	        SELECT 1 FROM agent_connector_credentials acc
	        WHERE acc.agent_id = ac.agent_id
	          AND acc.approver_id = ac.approver_id
	          AND acc.connector_id = ac.connector_id
	          AND acc.connector_instance_id = ac.connector_instance_id
	      )
	  )`,

	// Drop non-default instances that no longer have a binding. This matches
	// the settings UI, which deletes those orphans on load.
	`DELETE FROM agent_connectors
	WHERE is_default = 0
	  AND NOT EXISTS (
	    SELECT 1 FROM agent_connector_credentials acc
	    WHERE acc.agent_id = agent_connectors.agent_id
	      AND acc.approver_id = agent_connectors.approver_id
	      AND acc.connector_id = agent_connectors.connector_id
	      AND acc.connector_instance_id = agent_connectors.connector_instance_id
	  )`,
}

// RepairDanglingConnectorInstances deletes stale connector instances and
// reassigns the default when a deleted credential left it unbound. Returns the
// number of rows changed. Safe to run repeatedly.
func RepairDanglingConnectorInstances(ctx context.Context, d DBTX) (int64, error) {
	tx, owned, err := BeginOrContinue(ctx, d)
	if err != nil {
		return 0, err
	}
	if owned {
		defer RollbackTx(ctx, tx) //nolint:errcheck // commit path clears
	}

	var changed int64
	for _, stmt := range danglingConnectorRepairSQL {
		res, err := tx.Exec(ctx, stmt)
		if err != nil {
			return 0, fmt.Errorf("repair dangling connector instances: %w", err)
		}
		changed += RowsAffected(res)
	}
	if owned {
		if err := CommitTx(ctx, tx); err != nil {
			return 0, err
		}
	}
	return changed, nil
}

// sqlDBTX adapts *sql.DB or *sql.Tx to DBTX. It does not start transactions;
// callers that need atomicity should pass a *sql.Tx.
type sqlDBTX struct {
	exec     func(ctx context.Context, query string, args ...any) (sql.Result, error)
	query    func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	queryRow func(ctx context.Context, query string, args ...any) *sql.Row
}

func (s sqlDBTX) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.exec(ctx, query, args...)
}
func (s sqlDBTX) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.query(ctx, query, args...)
}
func (s sqlDBTX) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.queryRow(ctx, query, args...)
}

func repairDanglingOnSQLDB(ctx context.Context, sqlDB *sql.DB) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin connector instance repair: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // commit path clears

	adapted := sqlDBTX{
		exec:     tx.ExecContext,
		query:    tx.QueryContext,
		queryRow: tx.QueryRowContext,
	}
	n, err := RepairDanglingConnectorInstances(ctx, adapted)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit connector instance repair: %w", err)
	}
	if n > 0 {
		log.Printf("database: repaired stale connector instances (%d row changes)", n)
	}
	return nil
}
