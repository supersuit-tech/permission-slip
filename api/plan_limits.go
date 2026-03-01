package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// LimitExceededResponse builds the standard error response for plan limit violations.
// The response includes the error code, a human-readable upgrade message, and
// details with the current count and limit for client-side display.
func LimitExceededResponse(code ErrorCode, resource string, currentCount, limit int) ErrorResponse {
	resp := Forbidden(code, fmt.Sprintf("Free tier allows up to %d %s. Upgrade your plan to add more.", limit, resource))
	resp.Error.Details = map[string]any{
		"current_count": currentCount,
		"limit":         limit,
	}
	return resp
}

// checkAgentLimit verifies the user hasn't exceeded their plan's agent limit.
// Returns true if the request should be aborted (limit exceeded or error).
// When the plan has no limit (nil), this always returns false (allowed).
func checkAgentLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkAgentLimit: get subscription: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}
	if sp == nil || sp.Plan.MaxAgents == nil {
		return false // no subscription or unlimited plan
	}

	count, err := db.CountRegisteredAgentsByUser(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkAgentLimit: count agents: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}

	limit := *sp.Plan.MaxAgents
	if count >= limit {
		RespondError(w, r, http.StatusForbidden, LimitExceededResponse(ErrAgentLimitReached, "agents", count, limit))
		return true
	}
	return false
}

// checkStandingApprovalLimit verifies the user hasn't exceeded their plan's standing approval limit.
// Returns true if the request should be aborted (limit exceeded or error).
func checkStandingApprovalLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkStandingApprovalLimit: get subscription: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}
	if sp == nil || sp.Plan.MaxStandingApprovals == nil {
		return false
	}

	count, err := db.CountActiveStandingApprovalsByUser(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkStandingApprovalLimit: count standing approvals: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}

	limit := *sp.Plan.MaxStandingApprovals
	if count >= limit {
		RespondError(w, r, http.StatusForbidden, LimitExceededResponse(ErrStandingApprovalLimitReached, "active standing approvals", count, limit))
		return true
	}
	return false
}

// checkCredentialLimit verifies the user hasn't exceeded their plan's credential limit.
// Returns true if the request should be aborted (limit exceeded or error).
func checkCredentialLimit(ctx context.Context, w http.ResponseWriter, r *http.Request, d db.DBTX, userID string) bool {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkCredentialLimit: get subscription: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}
	if sp == nil || sp.Plan.MaxCredentials == nil {
		return false
	}

	count, err := db.CountCredentialsByUser(ctx, d, userID)
	if err != nil {
		log.Printf("[%s] checkCredentialLimit: count credentials: %v", TraceID(r.Context()), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check plan limits"))
		return true
	}

	limit := *sp.Plan.MaxCredentials
	if count >= limit {
		RespondError(w, r, http.StatusForbidden, LimitExceededResponse(ErrCredentialLimitReached, "stored credentials", count, limit))
		return true
	}
	return false
}
