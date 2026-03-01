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
	ID                   string
	UserID               string
	PlanID               string
	Status               SubscriptionStatus
	StripeCustomerID     *string // nil for free-tier users (no Stripe setup)
	StripeSubscriptionID *string // nil for free-tier users
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

const subscriptionColumns = `id, user_id, plan_id, status, stripe_customer_id, stripe_subscription_id, current_period_start, current_period_end, created_at, updated_at`

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
func UpdateSubscriptionPlan(ctx context.Context, db DBTX, userID, planID string) (*Subscription, error) {
	s, err := scanSubscription(db.QueryRow(ctx,
		`UPDATE subscriptions
		 SET plan_id = $2, updated_at = now()
		 WHERE user_id = $1
		 RETURNING `+subscriptionColumns,
		userID, planID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
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
	tag, err := db.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plan_id)
		 SELECT p.id, $1
		 FROM profiles p
		 LEFT JOIN subscriptions s ON s.user_id = p.id
		 WHERE s.id IS NULL`,
		defaultPlan)
	if err != nil {
		return 0, err
	}
	total += tag.RowsAffected()

	// Step 2: When billing is disabled, upgrade "free" subscriptions to the
	// unlimited plan. This handles users backfilled by the initial migration
	// (which always assigns "free") before BILLING_ENABLED existed.
	if !billingEnabled {
		tag, err = db.Exec(ctx,
			`UPDATE subscriptions SET plan_id = $1, updated_at = now()
			 WHERE plan_id = 'free'`,
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

// GetSubscriptionWithPlan returns the user's subscription joined with plan
// details, or nil if the user has no subscription.
func GetSubscriptionWithPlan(ctx context.Context, db DBTX, userID string) (*SubscriptionWithPlan, error) {
	var sp SubscriptionWithPlan
	err := db.QueryRow(ctx,
		`SELECT s.id, s.user_id, s.plan_id, s.status,
		        s.stripe_customer_id, s.stripe_subscription_id,
		        s.current_period_start, s.current_period_end,
		        s.created_at, s.updated_at,
		        p.id, p.name, p.max_requests_per_month, p.max_agents,
		        p.max_standing_approvals, p.max_credentials,
		        p.audit_retention_days, p.price_per_request_millicents,
		        p.created_at
		 FROM subscriptions s
		 JOIN plans p ON p.id = s.plan_id
		 WHERE s.user_id = $1`,
		userID,
	).Scan(
		&sp.ID,
		&sp.UserID,
		&sp.PlanID,
		&sp.Status,
		&sp.StripeCustomerID,
		&sp.StripeSubscriptionID,
		&sp.CurrentPeriodStart,
		&sp.CurrentPeriodEnd,
		&sp.CreatedAt,
		&sp.UpdatedAt,
		&sp.Plan.ID,
		&sp.Plan.Name,
		&sp.Plan.MaxRequestsPerMonth,
		&sp.Plan.MaxAgents,
		&sp.Plan.MaxStandingApprovals,
		&sp.Plan.MaxCredentials,
		&sp.Plan.AuditRetentionDays,
		&sp.Plan.PricePerRequestMillicents,
		&sp.Plan.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sp, nil
}
