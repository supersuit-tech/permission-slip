package api

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// resourceLimitConfig holds the parameters for a single resource limit check.
type resourceLimitConfig struct {
	errorCode    ErrorCode
	resourceName string                                                        // human-readable, e.g. "agents"
	getLimit     func(p *db.Plan) *int                                         // extracts the plan limit (nil = unlimited)
	countFn      func(ctx context.Context, d db.DBTX, uid string) (int, error) // counts the user's current resources
}

// checkResourceLimit is the generic implementation for all plan-based resource limits.
// Returns true if the request should be aborted (limit exceeded or internal error).
func checkResourceLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string, cfg resourceLimitConfig) bool {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkResourceLimit(%s): get subscription: %v", TraceID(r.Context()), cfg.resourceName, err)
		CaptureError(r.Context(), fmt.Errorf("check %s limit: get subscription: %w", cfg.resourceName, err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}
	if sp == nil {
		return false // no subscription — bypass limits
	}

	limit := cfg.getLimit(&sp.Plan)
	if limit == nil {
		return false // unlimited plan
	}

	count, err := cfg.countFn(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkResourceLimit(%s): count: %v", TraceID(r.Context()), cfg.resourceName, err)
		CaptureError(r.Context(), fmt.Errorf("check %s limit: count: %w", cfg.resourceName, err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}

	if count >= *limit {
		resp := Forbidden(cfg.errorCode, fmt.Sprintf(
			"%s plan allows up to %d %s. Upgrade your plan to add more.",
			sp.Plan.Name, *limit, cfg.resourceName,
		))
		resp.Error.Details = map[string]any{
			"current_count": count,
			"limit":         *limit,
			"plan_id":       sp.Plan.ID,
		}
		RespondError(w, r, http.StatusForbidden, resp)
		return true
	}
	return false
}

// quotaReservedKey is a context key indicating that checkRequestQuota already
// atomically incremented usage_periods.request_count. When set, the audit
// metering path should only update the breakdown (not re-increment the count).
type quotaCtxKey struct{}

// WithQuotaReserved returns a new context marked as having already reserved
// a quota slot via atomic increment.
func WithQuotaReserved(ctx context.Context) context.Context {
	return context.WithValue(ctx, quotaCtxKey{}, true)
}

// IsQuotaReserved returns true if the request count was already atomically
// incremented by checkRequestQuota.
func IsQuotaReserved(ctx context.Context) bool {
	v, _ := ctx.Value(quotaCtxKey{}).(bool)
	return v
}

// checkRequestQuota verifies the user hasn't exceeded their plan's monthly
// request quota. For free-tier users, the count is atomically incremented as
// part of the check to prevent TOCTOU races under concurrent requests. Returns
// the (possibly updated) request and true if the request should be aborted.
func checkRequestQuota(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) (*http.Request, bool) {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkRequestQuota: get subscription: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), fmt.Errorf("check request quota: get subscription: %w", err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return r, true
	}
	if sp == nil {
		return r, false // no subscription — bypass limits
	}

	limit := sp.Plan.MaxRequestsPerMonth
	if limit == nil {
		return r, false // unlimited plan (paid tier)
	}

	now := time.Now()
	periodStart, periodEnd := db.BillingPeriodBounds(now)
	resetAt := periodEnd.UTC().Format(time.RFC3339)

	// Atomically try to reserve a quota slot. This increments request_count
	// only if it's currently below the limit, preventing TOCTOU races where
	// concurrent requests could each pass a read-then-check before any
	// increment is recorded.
	reserved, err := db.ReserveRequestQuota(ctx, d, userID, periodStart, periodEnd, *limit)
	if err != nil {
		log.Printf("[%s] checkRequestQuota: reserve quota: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), fmt.Errorf("check request quota: reserve: %w", err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return r, true
	}

	if !reserved {
		// Quota exhausted. Fetch current count for the error response.
		usage, _ := db.GetCurrentPeriodUsage(ctx, d, userID)
		currentCount := *limit
		if usage != nil {
			currentCount = usage.RequestCount
		}

		retryAfter := int(math.Ceil(time.Until(periodEnd).Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}

		resp := QuotaExceeded(
			fmt.Sprintf("Free tier limit of %d requests/month reached. Upgrade to continue.", *limit),
			retryAfter,
		)
		resp.Error.Details = map[string]any{
			"current_usage": currentCount,
			"limit":         *limit,
			"plan_id":       sp.Plan.ID,
			"reset_at":      resetAt,
		}
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		RespondError(w, r, http.StatusTooManyRequests, resp)
		return r, true
	}

	// Reservation succeeded. Mark context so audit metering only updates
	// the breakdown without re-incrementing the count.
	r = r.WithContext(WithQuotaReserved(r.Context()))

	// Fetch the post-increment count for informational headers.
	usage, _ := db.GetCurrentPeriodUsage(ctx, d, userID)
	currentCount := 1
	if usage != nil {
		currentCount = usage.RequestCount
	}

	// Set informational quota headers on allowed requests so SDK authors
	// can proactively warn users approaching their limit (follows the
	// pattern of GitHub, Stripe, and OpenAI APIs).
	remaining := *limit - currentCount
	if remaining < 0 {
		remaining = 0
	}
	w.Header().Set("X-Quota-Limit", strconv.Itoa(*limit))
	w.Header().Set("X-Quota-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-Quota-Reset", resetAt)

	return r, false
}

// checkAgentLimit verifies the user hasn't exceeded their plan's agent limit.
// Returns true if the request should be aborted (limit exceeded or error).
func checkAgentLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	return checkResourceLimit(ctx, w, r, d, userID, resourceLimitConfig{
		errorCode:    ErrAgentLimitReached,
		resourceName: "agents",
		getLimit:     func(p *db.Plan) *int { return p.MaxAgents },
		countFn:      db.CountRegisteredAgentsByUser,
	})
}

// checkStandingApprovalLimit verifies the user hasn't exceeded their plan's standing approval limit.
// Returns true if the request should be aborted (limit exceeded or error).
func checkStandingApprovalLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	return checkResourceLimit(ctx, w, r, d, userID, resourceLimitConfig{
		errorCode:    ErrStandingApprovalLimitReached,
		resourceName: "active standing approvals",
		getLimit:     func(p *db.Plan) *int { return p.MaxStandingApprovals },
		countFn:      db.CountActiveStandingApprovalsByUser,
	})
}

// checkCredentialLimit verifies the user hasn't exceeded their plan's credential limit.
// Returns true if the request should be aborted (limit exceeded or error).
func checkCredentialLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	return checkResourceLimit(ctx, w, r, d, userID, resourceLimitConfig{
		errorCode:    ErrCredentialLimitReached,
		resourceName: "stored credentials",
		getLimit:     func(p *db.Plan) *int { return p.MaxCredentials },
		countFn:      db.CountCredentialsByUser,
	})
}
