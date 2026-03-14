package db

import (
	"context"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/config"
)

// PurgeExpiredAuditEvents deletes audit events older than the retention period
// for each user's plan. Free-tier users retain 7 days; paid users retain 90 days.
//
// A 7-day grace period is applied after a downgrade: if a user's subscription
// has downgraded_at set within the last 7 days, the previous paid plan's
// retention (90 days) is used instead of the current plan's shorter retention.
//
// Users without a subscription row (shouldn't happen, but defensive) are treated
// as free-tier and get the default retention.
//
// Plan retention days are read from config/plans.json — no plans table join needed.
// The query uses a CASE expression to map plan_id → retention days inline.
func PurgeExpiredAuditEvents(ctx context.Context, db DBTX) (int64, error) {
	// Build the retention CASE expression from config.
	plans := config.AllPlans()
	if len(plans) == 0 {
		return 0, fmt.Errorf("no plans found in config")
	}

	// Default to the free plan's retention for any unknown plan_id.
	freePlan := config.GetPlan(config.PlanFree)
	defaultRetention := 7
	if freePlan != nil {
		defaultRetention = freePlan.AuditRetentionDays
	}

	// Build "CASE s.plan_id WHEN 'free' THEN 7 WHEN 'pay_as_you_go' THEN 90 ELSE 7 END"
	caseExpr := "CASE s.plan_id"
	for _, p := range plans {
		caseExpr += fmt.Sprintf(" WHEN '%s' THEN %d", p.ID, p.AuditRetentionDays)
	}
	caseExpr += fmt.Sprintf(" ELSE %d END", defaultRetention)

	// Derive grace period days from the DowngradeGracePeriod constant so both
	// the Go logic and the SQL query stay in sync automatically.
	gracePeriodDays := int(DowngradeGracePeriod.Hours() / 24)

	// Pass 1: Purge events for users with a subscription, using their plan's
	// retention period. During the downgrade grace period, use the paid plan's
	// retention instead so users have time to export data.
	tag1, err := db.Exec(ctx, fmt.Sprintf(`
		DELETE FROM audit_events ae
		USING subscriptions s
		WHERE ae.user_id = s.user_id
		  AND ae.created_at < now() - make_interval(days =>
		      CASE WHEN s.downgraded_at IS NOT NULL
		                AND s.downgraded_at > now() - interval '%d days'
		           THEN %d
		           ELSE %s
		      END)`, gracePeriodDays, PaidPlanRetentionDays(), caseExpr))
	if err != nil {
		return 0, fmt.Errorf("purge expired audit events (subscribed users): %w", err)
	}

	// Pass 2: Purge events for users without a subscription row, using the
	// free-tier default. This is defensive — every user should have
	// a subscription, but we don't want orphaned events to accumulate.
	tag2, err := db.Exec(ctx, fmt.Sprintf(`
		DELETE FROM audit_events ae
		WHERE NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = ae.user_id)
		  AND ae.created_at < now() - interval '%d days'`, defaultRetention))
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
