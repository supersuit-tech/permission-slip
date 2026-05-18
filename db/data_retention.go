package db

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AuditRetentionDaysFromEnv returns AUDIT_RETENTION_DAYS (default 90, min 1, max 3650).
func AuditRetentionDaysFromEnv() int {
	const def = 90
	const maxDays = 3650
	v := strings.TrimSpace(os.Getenv("AUDIT_RETENTION_DAYS"))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	if n > maxDays {
		return maxDays
	}
	return n
}

// PurgeExpiredAuditEvents deletes audit events older than retentionDays for all users.
func PurgeExpiredAuditEvents(ctx context.Context, db DBTX, retentionDays int) (int64, error) {
	if retentionDays < 1 {
		retentionDays = 90
	}
	tag, err := db.Exec(ctx,
		`DELETE FROM audit_events WHERE created_at < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-' || $1 || ' days')`,
		retentionDays)
	if err != nil {
		return 0, fmt.Errorf("purge expired audit events: %w", err)
	}
	return RowsAffected(tag), nil
}

// DeleteAccount deletes a user and all associated data. Deleting the users row
// cascades to profiles, which cascades to agents, approvals, credentials,
// standing approvals, audit events, etc.
//
// Vault secrets (encrypted credentials) are stored in vault_secrets outside
// the FK cascade, so they must be deleted separately before the user row is
// removed. Pass a nil vaultDeleteFn if no vault cleanup is needed (e.g. in tests).
func DeleteAccount(ctx context.Context, d DBTX, userID string, vaultDeleteFn func(ctx context.Context, tx DBTX, secretID string) error) error {
	// Step 1: Delete vault secrets for all user credentials.
	if vaultDeleteFn != nil {
		rows, err := d.Query(ctx,
			`SELECT vault_secret_id FROM credentials WHERE user_id = $1`, userID)
		if err != nil {
			return fmt.Errorf("list credential vault secrets: %w", err)
		}
		defer rows.Close()

		var secretIDs []string
		for rows.Next() {
			var sid string
			if err := rows.Scan(&sid); err != nil {
				return fmt.Errorf("scan vault secret id: %w", err)
			}
			secretIDs = append(secretIDs, sid)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate vault secret ids: %w", err)
		}

		for _, sid := range secretIDs {
			if err := vaultDeleteFn(ctx, d, sid); err != nil {
				return fmt.Errorf("delete vault secret %s: %w", sid, err)
			}
		}
	}

	// Step 2: Delete the user row. CASCADE removes auth_sessions, profiles,
	// and all downstream child rows (agents, approvals, credentials, etc.).
	tag, err := d.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if RowsAffected(tag) == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
