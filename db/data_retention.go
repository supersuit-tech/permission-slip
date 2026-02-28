package db

import (
	"context"
	"fmt"
)

// PurgeExpiredAuditEvents deletes audit events older than the retention period
// for each user's plan. Free-tier users retain 7 days; paid users retain 90 days.
//
// Users without a subscription row (shouldn't happen, but defensive) are treated
// as free-tier and get the 7-day default retention.
//
// Returns the total number of rows deleted.
func PurgeExpiredAuditEvents(ctx context.Context, db DBTX) (int64, error) {
	// Pass 1: Purge events for users with a subscription, using their plan's
	// retention period.
	tag1, err := db.Exec(ctx, `
		DELETE FROM audit_events ae
		USING subscriptions s
		JOIN plans p ON p.id = s.plan_id
		WHERE ae.user_id = s.user_id
		  AND ae.created_at < now() - make_interval(days => p.audit_retention_days)`)
	if err != nil {
		return 0, fmt.Errorf("purge expired audit events (subscribed users): %w", err)
	}

	// Pass 2: Purge events for users without a subscription row, using the
	// free-tier default (7 days). This is defensive — every user should have
	// a subscription, but we don't want orphaned events to accumulate.
	tag2, err := db.Exec(ctx, `
		DELETE FROM audit_events ae
		WHERE NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = ae.user_id)
		  AND ae.created_at < now() - interval '7 days'`)
	if err != nil {
		return tag1.RowsAffected(), fmt.Errorf("purge expired audit events (unsubscribed users): %w", err)
	}

	return tag1.RowsAffected() + tag2.RowsAffected(), nil
}

// DeleteAccount deletes a user's profile and all associated data. Because most
// child tables use ON DELETE CASCADE, deleting the profile row removes agents,
// approvals, credentials, standing approvals, subscriptions, audit events, etc.
//
// Vault secrets (encrypted credentials) are stored outside the FK graph in
// Supabase Vault's vault.secrets table, so they must be deleted separately
// before the profile row is removed. Pass a nil vaultDeleteFn if no vault
// cleanup is needed (e.g. in tests).
//
// The caller is responsible for deleting the Supabase auth.users row (via
// Supabase Admin API) after this function succeeds.
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

	// Step 2: Delete the profile row. ON DELETE CASCADE removes all child rows
	// (agents, approvals, credentials, subscriptions, audit_events, etc.).
	tag, err := d.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("profile not found")
	}
	return nil
}
