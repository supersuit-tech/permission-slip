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

// checkRequestQuota verifies the user hasn't exceeded their plan's monthly request quota.
// Unlike resource limits (which return 403), this returns 429 with a Retry-After header
// indicating seconds until the billing period resets.
// Returns true if the request should be aborted (quota exceeded or internal error).
func checkRequestQuota(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkRequestQuota: get subscription: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), fmt.Errorf("check request quota: get subscription: %w", err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}
	if sp == nil {
		return false // no subscription — bypass limits
	}

	limit := sp.Plan.MaxRequestsPerMonth
	if limit == nil {
		return false // unlimited plan (paid tier)
	}

	now := time.Now()
	usage, err := db.GetCurrentPeriodUsage(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkRequestQuota: get usage: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), fmt.Errorf("check request quota: get usage: %w", err))
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}

	currentCount := 0
	if usage != nil {
		currentCount = usage.RequestCount
	}

	if currentCount >= *limit {
		_, periodEnd := db.BillingPeriodBounds(now)
		retryAfter := int(math.Ceil(time.Until(periodEnd).Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}

		resp := ErrorResponse{Error: Error{
			Code:       ErrMonthlyQuotaExceeded,
			Message:    fmt.Sprintf("Free tier limit of %d requests/month reached. Upgrade to continue.", *limit),
			Retryable:  true,
			RetryAfter: retryAfter,
			Details: map[string]any{
				"current_usage": currentCount,
				"limit":         *limit,
			},
		}}
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		RespondError(w, r, http.StatusTooManyRequests, resp)
		return true
	}
	return false
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
