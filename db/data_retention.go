package db

import (
	"context"
	"fmt"
	"strings"
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
// Plan retention days are read from plans_embedded.json — no plans table join needed.
// The query uses a CASE expression to map plan_id → retention days inline.
func PurgeExpiredAuditEvents(ctx context.Context, db DBTX) (int64, error) {
	// Build the retention CASE expression from embedded plan definitions.
	plans := AllPlans()
	if len(plans) == 0 {
		return 0, fmt.Errorf("no plans found in config")
	}

	// Default to the free plan's retention for any unknown plan_id.
	freePlan := GetPlan(PlanFree)
	defaultRetention := 7
	if freePlan != nil {
		defaultRetention = freePlan.AuditRetentionDays
	}

	// Build parameterized VALUES list for plan_id → retention_days mapping.
	// Uses ($1, $2), ($3, $4), ... to avoid interpolating
	// plan IDs into SQL strings.
	var valuesClauses []string
	var args []any
	paramIdx := 1
	for _, p := range plans {
		valuesClauses = append(valuesClauses,
			fmt.Sprintf("($%d, $%d)", paramIdx, paramIdx+1))
		args = append(args, p.ID, p.AuditRetentionDays)
		paramIdx += 2
	}

	// Derive grace period days from the DowngradeGracePeriod constant so both
	// the Go logic and the SQL query stay in sync automatically.
	gracePeriodDays := int(DowngradeGracePeriod.Hours() / 24)

	// Append remaining parameters: grace period, paid retention, default retention.
	gracePeriodParam := paramIdx
	paidRetentionParam := paramIdx + 1
	defaultRetentionParam := paramIdx + 2
	args = append(args, gracePeriodDays, PaidPlanRetentionDays(), defaultRetention)

	// Pass 1: Purge events for users with a subscription, using their plan's
	// retention period. During the downgrade grace period, use the paid plan's
	// retention instead so users have time to export data.
	//
	// SQLite doesn't support DELETE...USING, so we use a CTE + subquery.
	// strftime %% escaping is required because this string is built via fmt.Sprintf.
	query := fmt.Sprintf(`
		WITH plan_retention(plan_id, retention_days) AS (VALUES %s)
		DELETE FROM audit_events WHERE id IN (
		    SELECT ae.id FROM audit_events ae
		    JOIN subscriptions s ON ae.user_id = s.user_id
		    LEFT JOIN plan_retention ON s.plan_id = plan_retention.plan_id
		    WHERE ae.created_at < strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ', 'now', '-' || (
		        CASE WHEN s.downgraded_at IS NOT NULL
		                  AND s.downgraded_at > strftime('%%Y-%%m-%%dT%%H:%%M:%%fZ', 'now', '-' || $%d || ' days')
		             THEN $%d
		             ELSE COALESCE(plan_retention.retention_days, $%d)
		        END) || ' days')
		)`,
		strings.Join(valuesClauses, ", "),
		gracePeriodParam,
		paidRetentionParam,
		defaultRetentionParam,
	)

	tag1, err := db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("purge expired audit events (subscribed users): %w", err)
	}

	// Pass 2: Purge events for users without a subscription row, using the
	// free-tier default. This is defensive — every user should have
	// a subscription, but we don't want orphaned events to accumulate.
	tag2, err := db.Exec(ctx,
		`DELETE FROM audit_events
		 WHERE NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = audit_events.user_id)
		   AND created_at < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-' || $1 || ' days')`,
		defaultRetention)
	if err != nil {
		return RowsAffected(tag1), fmt.Errorf("purge expired audit events (unsubscribed users): %w", err)
	}

	return RowsAffected(tag1) + RowsAffected(tag2), nil
}

// DeleteAccount deletes a user and all associated data. Deleting the users row
// cascades to profiles, which cascades to agents, approvals, credentials,
// standing approvals, subscriptions, audit events, etc.
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
