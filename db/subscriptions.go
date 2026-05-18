package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
)

// validSubscriptionStatuses is the set of allowed subscription statuses,
// mirroring the CHECK constraint in the subscriptions table.
var validSubscriptionStatuses = map[SubscriptionStatus]bool{
	SubscriptionStatusActive:    true,
	SubscriptionStatusPastDue:   true,
	SubscriptionStatusCancelled: true,
}

// IsValidSubscriptionStatus checks if the given status is valid.
func IsValidSubscriptionStatus(s SubscriptionStatus) bool {
	return validSubscriptionStatuses[s]
}

// Subscription represents a row from the subscriptions table.
// Each user has at most one subscription (enforced by UNIQUE on user_id).
// Billing periods are aligned to calendar months (via date_trunc defaults).
type Subscription struct {
	ID                     string
	UserID                 string
	PlanID                 string
	Status                 SubscriptionStatus
	StripeCustomerID       *string // nil for free-tier users (no Stripe setup)
	StripeSubscriptionID   *string // nil for free-tier users
	CurrentPeriodStart     time.Time
	CurrentPeriodEnd       time.Time
	DowngradedAt           *time.Time // set when plan changes from paid to free; nil otherwise
	QuotaPlanID            *string    // plan whose quotas apply during post-downgrade grace; nil when not in quota grace
	QuotaEntitlementsUntil *time.Time // paid quotas apply until this instant (exclusive convention matches billing period end)
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

const subscriptionColumns = `id, user_id, plan_id, status, stripe_customer_id, stripe_subscription_id, current_period_start, current_period_end, downgraded_at, quota_plan_id, quota_entitlements_until, created_at, updated_at`

func scanSubscription(row rowScanner) (*Subscription, error) {
	var s Subscription
	var curStart, curEnd, createdAt, updatedAt sql.NullString
	var downgradedAt, quotaUntil sql.NullString
	err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.PlanID,
		&s.Status,
		&s.StripeCustomerID,
		&s.StripeSubscriptionID,
		&curStart,
		&curEnd,
		&downgradedAt,
		&s.QuotaPlanID,
		&quotaUntil,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}
	var err2 error
	s.CurrentPeriodStart, err2 = sqliteTimeRequired(curStart)
	if err2 != nil {
		return nil, err2
	}
	s.CurrentPeriodEnd, err2 = sqliteTimeRequired(curEnd)
	if err2 != nil {
		return nil, err2
	}
	s.DowngradedAt, err2 = sqliteTimePtr(downgradedAt)
	if err2 != nil {
		return nil, err2
	}
	s.QuotaEntitlementsUntil, err2 = sqliteTimePtr(quotaUntil)
	if err2 != nil {
		return nil, err2
	}
	s.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	s.UpdatedAt, err2 = sqliteTimeRequired(updatedAt)
	if err2 != nil {
		return nil, err2
	}
	return &s, nil
}

// GetSubscriptionByUserID returns the subscription for the given user, or nil
// if the user has no subscription.
func GetSubscriptionByUserID(ctx context.Context, db DBTX, userID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE user_id = $1", userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// CreateSubscription inserts a new subscription and returns it.
func CreateSubscription(ctx context.Context, db DBTX, userID, planID string) (*Subscription, error) {
	id := uuid.NewString()
	return scanSubscription(db.QueryRow(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id)
		 VALUES ($1, $2, $3)
		 RETURNING `+subscriptionColumns,
		id, userID, planID))
}

// UpdateSubscriptionPlan changes the plan for a user's subscription.
// When downgrading (moving to a plan with shorter retention), sets downgraded_at
// to trigger a grace period before the shorter retention window takes effect.
// When upgrading, clears downgraded_at since the longer retention applies immediately.
// When moving to a paid plan, clears quota grace columns.
// For pay-as-you-go → free with paid quotas until period end, use
// DowngradeSubscriptionToFreeWithQuotaGrace instead.
func UpdateSubscriptionPlan(ctx context.Context, db DBTX, userID, planID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET downgraded_at = CASE
		         WHEN $2 = 'free' AND plan_id != 'free' THEN strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		         WHEN $2 != 'free' THEN NULL
		         ELSE downgraded_at
		     END,
		     quota_plan_id = CASE WHEN $2 != 'free' THEN NULL ELSE quota_plan_id END,
		     quota_entitlements_until = CASE WHEN $2 != 'free' THEN NULL ELSE quota_entitlements_until END,
		     plan_id = $2,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, planID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// DowngradeSubscriptionToFreeWithQuotaGrace sets plan to free, records
// downgraded_at, and snapshots paid quota entitlements until periodEnd.
// paidPlanID must be the plan the user is leaving (e.g. PlanPayAsYouGo).
func DowngradeSubscriptionToFreeWithQuotaGrace(ctx context.Context, db DBTX, userID, paidPlanID string, periodEnd time.Time) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET plan_id = 'free',
		     downgraded_at = CASE WHEN plan_id != 'free' THEN strftime('%Y-%m-%dT%H:%M:%fZ', 'now') ELSE downgraded_at END,
		     quota_plan_id = COALESCE(quota_plan_id, $2),
		     quota_entitlements_until = COALESCE(quota_entitlements_until, $3),
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, paidPlanID, TimestampForSQLite(periodEnd)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// ApplyStripeSubscriptionDeletedToFree downgrades to free (or keeps free_pro),
// syncs status, and ensures quota grace columns are set from Stripe's period
// end when missing (idempotent with user-initiated downgrade).
func ApplyStripeSubscriptionDeletedToFree(ctx context.Context, db DBTX, userID string, targetPlan string, nextStatus SubscriptionStatus, quotaPlanID *string, periodEnd *time.Time) (*Subscription, error) {
	var qPlan any
	if quotaPlanID != nil {
		qPlan = *quotaPlanID
	}
	var qEnd any
	if periodEnd != nil {
		qEnd = TimestampForSQLite(*periodEnd)
	}
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET plan_id = $2,
		     status = $3,
		     stripe_subscription_id = CASE WHEN $2 = 'free' THEN NULL ELSE stripe_subscription_id END,
		     downgraded_at = CASE
		             WHEN $2 = 'free' AND plan_id != 'free' THEN strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		             WHEN $2 = 'free' THEN downgraded_at
		             ELSE NULL
		         END,
		     quota_plan_id = CASE
		             WHEN $4 IS NOT NULL THEN COALESCE(quota_plan_id, $4)
		             ELSE quota_plan_id
		         END,
		     quota_entitlements_until = CASE
		             WHEN $5 IS NOT NULL THEN COALESCE(quota_entitlements_until, $5)
		             ELSE quota_entitlements_until
		         END,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, targetPlan, nextStatus, qPlan, qEnd))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UpgradeSubscriptionPlan atomically upgrades a subscription to a new plan,
// but only if the user is currently on the expected old plan. This prevents
// race conditions where two concurrent checkout webhooks could both upgrade
// the same user. Returns nil (no error) if the user's current plan doesn't
// match expectedOldPlanID (i.e., the upgrade was already applied).
func UpgradeSubscriptionPlan(ctx context.Context, db DBTX, userID, expectedOldPlanID, newPlanID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET plan_id = $3,
		     downgraded_at = NULL,
		     quota_plan_id = NULL,
		     quota_entitlements_until = NULL,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1 AND plan_id = $2
		 RETURNING `+subscriptionColumns,
		userID, expectedOldPlanID, newPlanID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil // already upgraded or plan changed — idempotent no-op
	}
	return s, err
}

// UpgradePayAsYouGoFromFreeOrFreePro upgrades to pay_as_you_go when the user is
// currently on free or free_pro. Used by Stripe checkout activation so comped
// users can still subscribe for paid billing if they choose.
func UpgradePayAsYouGoFromFreeOrFreePro(ctx context.Context, db DBTX, userID string) (*Subscription, error) {
	s, err := UpgradeSubscriptionPlan(ctx, db, userID, PlanFree, PlanPayAsYouGo)
	if err != nil {
		return nil, err
	}
	if s != nil {
		return s, nil
	}
	return UpgradeSubscriptionPlan(ctx, db, userID, PlanFreePro, PlanPayAsYouGo)
}

// UpdateSubscriptionStatus updates the status of a user's subscription.
// Returns an error if the status is not one of the allowed values.
func UpdateSubscriptionStatus(ctx context.Context, db DBTX, userID string, status SubscriptionStatus) (*Subscription, error) {
	if !IsValidSubscriptionStatus(status) {
		return nil, fmt.Errorf("invalid subscription status: %q", status)
	}
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET status = $2, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, status))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UpdateSubscriptionStripe sets the Stripe customer and subscription IDs.
func UpdateSubscriptionStripe(ctx context.Context, db DBTX, userID string, stripeCustomerID, stripeSubscriptionID *string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET stripe_customer_id = $2, stripe_subscription_id = $3, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, stripeCustomerID, stripeSubscriptionID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UpdateSubscriptionPeriod updates the billing period timestamps.
func UpdateSubscriptionPeriod(ctx context.Context, db DBTX, userID string, periodStart, periodEnd time.Time) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET current_period_start = $2, current_period_end = $3, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, TimestampForSQLite(periodStart), TimestampForSQLite(periodEnd)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// EnsureAllUsersSubscribed makes sure every user has a subscription on the
// correct default plan. It does two things:
//
//  1. Creates subscriptions for users that don't have one yet.
//  2. When billing is disabled, updates any existing "free" subscriptions to
//     "pay_as_you_go" so that users backfilled by older migrations (which
//     hard-coded the "free" plan) get unlimited access.
//
// Returns the total number of rows created or updated.
func EnsureAllUsersSubscribed(ctx context.Context, db DBTX) (int64, error) {
	defaultPlan := PlanPayAsYouGo
	var total int64

	// Step 1: Create subscriptions for users without one.
	tag, err := db.Exec(ctx,
		`INSERT INTO subscriptions (id, user_id, plan_id)
		 SELECT lower(hex(randomblob(16))), p.id, $1
		 FROM profiles p
		 WHERE NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = p.id)`,
		defaultPlan)
	if err != nil {
		return 0, err
	}
	total += RowsAffected(tag)

	// Upgrade legacy free-tier subscriptions to the unlimited plan.
	tag, err = db.Exec(ctx,
		`UPDATE subscriptions SET plan_id = $1, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE plan_id IN ('free', 'free_pro')`,
		PlanPayAsYouGo)
	if err != nil {
		return total, err
	}
	total += RowsAffected(tag)

	return total, nil
}

// GetSubscriptionByStripeCustomerID returns the subscription with the given
// Stripe Customer ID, or nil if not found. Used by webhook handlers.
func GetSubscriptionByStripeCustomerID(ctx context.Context, db DBTX, stripeCustomerID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE stripe_customer_id = $1", stripeCustomerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// GetSubscriptionByStripeSubscriptionID returns the subscription with the given
// Stripe Subscription ID, or nil if not found. Used by webhook handlers.
func GetSubscriptionByStripeSubscriptionID(ctx context.Context, db DBTX, stripeSubscriptionID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE stripe_subscription_id = $1", stripeSubscriptionID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// SubscriptionWithPlan combines a subscription with its associated plan details
// in a single query. This avoids the N+1 pattern of fetching subscription then plan.
type SubscriptionWithPlan struct {
	Subscription
	Plan Plan
}

// DowngradeGracePeriod is the duration after a downgrade during which the
// previous (longer) retention window is still honoured. During this period
// EffectiveRetentionDays continues to use PaidPlanRetentionDays so users
// have time to export their data before it becomes inaccessible.
const DowngradeGracePeriod = 7 * 24 * time.Hour // 7 days

// paidPlanRetentionDays is the retention window for the pay-as-you-go plan.
// Derived from config/plans.json so it stays in sync with the plan definition.
var paidPlanRetentionDays = func() int {
	p := GetPlan(PlanPayAsYouGo)
	if p != nil {
		return p.AuditRetentionDays
	}
	return 90 // fallback
}()

// PaidPlanRetentionDays returns the audit retention window for the paid plan.
// Used during the downgrade grace period when the plan's own retention is shorter.
func PaidPlanRetentionDays() int { return paidPlanRetentionDays }

// EffectiveRetentionDays returns the audit log retention window to enforce
// for this subscription. During the downgrade grace period the previous
// (longer) retention is used so users have time to export data.
func (sp *SubscriptionWithPlan) EffectiveRetentionDays() int {
	if sp.DowngradedAt != nil && time.Since(*sp.DowngradedAt) < DowngradeGracePeriod {
		return PaidPlanRetentionDays()
	}
	return sp.Plan.AuditRetentionDays
}

// GracePeriodEndsAt returns the timestamp when the downgrade grace period
// expires, or nil if no grace period is active. This helps the frontend
// show users when their extended retention will end.
func (sp *SubscriptionWithPlan) GracePeriodEndsAt() *time.Time {
	if sp.DowngradedAt != nil && time.Since(*sp.DowngradedAt) < DowngradeGracePeriod {
		t := sp.DowngradedAt.Add(DowngradeGracePeriod)
		return &t
	}
	return nil
}

// ClearSubscriptionQuotaGrace clears active paid-plan quota snapshot columns
// immediately (user already on free). Returns the updated row, or nil if the
// user had no active quota grace window.
func ClearSubscriptionQuotaGrace(ctx context.Context, db DBTX, userID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET quota_plan_id = NULL,
		     quota_entitlements_until = NULL,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		   AND quota_plan_id IS NOT NULL
		   AND quota_entitlements_until IS NOT NULL
		   AND datetime(quota_entitlements_until) > datetime('now')
		 RETURNING `+subscriptionColumns,
		userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// ClearExpiredSubscriptionQuotaGrace clears quota grace columns when the
// entitlement window has ended (lazy expiration on read paths).
func ClearExpiredSubscriptionQuotaGrace(ctx context.Context, db DBTX, userID string) error {
	_, err := db.Exec(ctx,
		`UPDATE subscriptions
		 SET quota_plan_id = NULL,
		     quota_entitlements_until = NULL,
		     updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE user_id = $1
		   AND quota_plan_id IS NOT NULL
		   AND quota_entitlements_until IS NOT NULL
		   AND datetime(quota_entitlements_until) <= datetime('now')`,
		userID)
	return err
}

// IsInQuotaGrace reports whether the subscription has an active
// post-downgrade quota grace window (paid quotas still apply).
func (s *Subscription) IsInQuotaGrace() bool {
	return s.QuotaPlanID != nil && s.QuotaEntitlementsUntil != nil && time.Now().Before(*s.QuotaEntitlementsUntil)
}

// EffectiveQuotaPlan returns the plan whose resource/request quotas should
// apply. During an active quota grace period after downgrade, this is the
// snapshotted paid plan; otherwise the current plan row.
func (sp *SubscriptionWithPlan) EffectiveQuotaPlan() *Plan {
	if sp.IsInQuotaGrace() {
		if p := GetPlan(*sp.QuotaPlanID); p != nil {
			return p
		}
	}
	return &sp.Plan
}

// GetSubscriptionWithPlan returns the user's subscription with plan details
// attached from config (no DB join needed), or nil if the user has no subscription.
func GetSubscriptionWithPlan(ctx context.Context, db DBTX, userID string) (*SubscriptionWithPlan, error) {
	sub, err := GetSubscriptionByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	if !sub.IsInQuotaGrace() && sub.QuotaPlanID != nil {
		if err := ClearExpiredSubscriptionQuotaGrace(ctx, db, userID); err != nil {
			return nil, err
		}
		sub, err = GetSubscriptionByUserID(ctx, db, userID)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, nil
		}
	}
	plan := GetPlan(sub.PlanID)
	if plan == nil {
		return nil, fmt.Errorf("plan %q not found in config", sub.PlanID)
	}
	return &SubscriptionWithPlan{
		Subscription: *sub,
		Plan:         *plan,
	}, nil
}
