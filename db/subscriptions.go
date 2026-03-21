package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
	ID                    string
	UserID                string
	PlanID                string
	Status                SubscriptionStatus
	StripeCustomerID      *string // nil for free-tier users (no Stripe setup)
	StripeSubscriptionID  *string // nil for free-tier users
	CurrentPeriodStart    time.Time
	CurrentPeriodEnd      time.Time
	DowngradedAt          *time.Time // set when plan changes from paid to free; nil otherwise
	QuotaPlanID           *string    // plan whose quotas apply during grace period; nil when not in grace
	QuotaEntitlementsUntil *time.Time // when quota grace expires; nil when not in grace
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

const subscriptionColumns = `id, user_id, plan_id, status, stripe_customer_id, stripe_subscription_id, current_period_start, current_period_end, downgraded_at, quota_plan_id, quota_entitlements_until, created_at, updated_at`

func scanSubscription(row pgx.Row) (*Subscription, error) {
	var s Subscription
	err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.PlanID,
		&s.Status,
		&s.StripeCustomerID,
		&s.StripeSubscriptionID,
		&s.CurrentPeriodStart,
		&s.CurrentPeriodEnd,
		&s.DowngradedAt,
		&s.QuotaPlanID,
		&s.QuotaEntitlementsUntil,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetSubscriptionByUserID returns the subscription for the given user, or nil
// if the user has no subscription.
func GetSubscriptionByUserID(ctx context.Context, db DBTX, userID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE user_id = $1", userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// CreateSubscription inserts a new subscription and returns it.
func CreateSubscription(ctx context.Context, db DBTX, userID, planID string) (*Subscription, error) {
	return scanSubscription(db.QueryRow(ctx,
		`INSERT INTO subscriptions (user_id, plan_id)
		 VALUES ($1, $2)
		 RETURNING `+subscriptionColumns,
		userID, planID))
}

// UpdateSubscriptionPlan changes the plan for a user's subscription.
// When downgrading (moving to a plan with shorter retention), sets downgraded_at
// to trigger a grace period before the shorter retention window takes effect.
// Also sets quota_plan_id and quota_entitlements_until so paid quotas are
// preserved until the end of the billing period the user already paid for.
// When upgrading, clears downgraded_at and quota columns since paid features apply immediately.
func UpdateSubscriptionPlan(ctx context.Context, db DBTX, userID, planID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET downgraded_at = CASE
		         WHEN $2 = 'free' AND plan_id != 'free' THEN now()
		         WHEN $2 != 'free' THEN NULL
		         ELSE downgraded_at
		     END,
		     quota_plan_id = CASE
		         WHEN $2 = 'free' AND plan_id != 'free' THEN plan_id
		         WHEN $2 != 'free' THEN NULL
		         ELSE quota_plan_id
		     END,
		     quota_entitlements_until = CASE
		         WHEN $2 = 'free' AND plan_id != 'free' THEN current_period_end
		         WHEN $2 != 'free' THEN NULL
		         ELSE quota_entitlements_until
		     END,
		     plan_id = $2,
		     updated_at = now()
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, planID))
	if errors.Is(err, pgx.ErrNoRows) {
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
		     updated_at = now()
		 WHERE user_id = $1 AND plan_id = $2
		 RETURNING `+subscriptionColumns,
		userID, expectedOldPlanID, newPlanID))
	if errors.Is(err, pgx.ErrNoRows) {
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
		 SET status = $2, updated_at = now()
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, status))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UpdateSubscriptionStripe sets the Stripe customer and subscription IDs.
func UpdateSubscriptionStripe(ctx context.Context, db DBTX, userID string, stripeCustomerID, stripeSubscriptionID *string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET stripe_customer_id = $2, stripe_subscription_id = $3, updated_at = now()
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, stripeCustomerID, stripeSubscriptionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// UpdateSubscriptionPeriod updates the billing period timestamps.
func UpdateSubscriptionPeriod(ctx context.Context, db DBTX, userID string, periodStart, periodEnd time.Time) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET current_period_start = $2, current_period_end = $3, updated_at = now()
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, periodStart, periodEnd))
	if errors.Is(err, pgx.ErrNoRows) {
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
func EnsureAllUsersSubscribed(ctx context.Context, db DBTX, billingEnabled bool) (int64, error) {
	defaultPlan := DefaultPlanID(billingEnabled)
	var total int64

	// Step 1: Create subscriptions for users without one.
	// FOR KEY SHARE prevents concurrent deletion of the profile row between
	// the SELECT and the INSERT, avoiding FK violations when parallel
	// transactions delete profiles (e.g. test cleanup).
	tag, err := db.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plan_id)
		 SELECT p.id, $1
		 FROM profiles p
		 WHERE NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = p.id)
		 FOR KEY SHARE OF p`,
		defaultPlan)
	if err != nil {
		return 0, err
	}
	total += tag.RowsAffected()

	// Step 2: When billing is disabled, upgrade free-tier subscriptions to the
	// unlimited plan. This handles users backfilled by the initial migration
	// (which always assigns "free") before BILLING_ENABLED existed, and
	// normalizes comped free_pro rows to pay_as_you_go for a single unlimited plan id.
	if !billingEnabled {
		tag, err = db.Exec(ctx,
			`UPDATE subscriptions SET plan_id = $1, updated_at = now()
			 WHERE plan_id IN ('free', 'free_pro')`,
			PlanPayAsYouGo)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
	}

	return total, nil
}

// GetSubscriptionByStripeCustomerID returns the subscription with the given
// Stripe Customer ID, or nil if not found. Used by webhook handlers.
func GetSubscriptionByStripeCustomerID(ctx context.Context, db DBTX, stripeCustomerID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE stripe_customer_id = $1", stripeCustomerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// GetSubscriptionByStripeSubscriptionID returns the subscription with the given
// Stripe Subscription ID, or nil if not found. Used by webhook handlers.
func GetSubscriptionByStripeSubscriptionID(ctx context.Context, db DBTX, stripeSubscriptionID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		"SELECT "+subscriptionColumns+" FROM subscriptions WHERE stripe_subscription_id = $1", stripeSubscriptionID))
	if errors.Is(err, pgx.ErrNoRows) {
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

// EffectiveQuotaPlan returns the plan whose resource limits should be enforced.
// During the quota grace period (after downgrade, before the paid billing period
// ends), this returns the previous paid plan so users keep paid quotas until the
// period they already paid for expires. Outside the grace period, returns the
// current plan.
func (sp *SubscriptionWithPlan) EffectiveQuotaPlan() *Plan {
	if sp.QuotaPlanID != nil && sp.QuotaEntitlementsUntil != nil {
		if time.Now().Before(*sp.QuotaEntitlementsUntil) {
			if p := GetPlan(*sp.QuotaPlanID); p != nil {
				return p
			}
		}
	}
	return &sp.Plan
}

// QuotaGracePeriodEndsAt returns the timestamp when the quota grace period
// expires, or nil if no quota grace period is active. This is separate from
// the audit retention grace period (GracePeriodEndsAt).
func (sp *SubscriptionWithPlan) QuotaGracePeriodEndsAt() *time.Time {
	if sp.QuotaPlanID != nil && sp.QuotaEntitlementsUntil != nil {
		if time.Now().Before(*sp.QuotaEntitlementsUntil) {
			return sp.QuotaEntitlementsUntil
		}
	}
	return nil
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
	plan := GetPlan(sub.PlanID)
	if plan == nil {
		return nil, fmt.Errorf("plan %q not found in config", sub.PlanID)
	}
	return &SubscriptionWithPlan{
		Subscription: *sub,
		Plan:         *plan,
	}, nil
}
