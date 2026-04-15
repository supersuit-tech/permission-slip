package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// DefaultVerifyRateLimiterConfig returns rate limit settings for the
// POST /agents/{id}/verify endpoint.
//
// This limit is per client IP, dedicated to the verify endpoint, and much
// tighter than the general pre-auth rate limiter so confirmation-code
// brute-force attempts are infeasible from a single origin regardless of
// how many agents the caller probes.
//
//	Per-IP: 0.5 req/s sustained, burst of 10.
//
// Per-agent throttling is not layered on top because the agent row's
// verification_attempts counter already locks the row after 5 wrong codes
// (see db.VerifyAgentRegistration and ErrVerificationLocked), which is a
// strictly stronger defense than a sliding rate limit on a single agent.
func DefaultVerifyRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		PerKeyRate:  0.5,
		PerKeyBurst: 10,
		GlobalRate:  10000,
		GlobalBurst: 10000,
	}
}

// checkVerifyAttemptRateLimit enforces per-IP throttling on
// POST /agents/{agent_id}/verify so confirmation-code brute-force attempts
// from a single origin are infeasible.
//
// Returns true when the request is allowed; otherwise it writes a 429 response
// with a Retry-After header and returns false. Skipped when deps.DevMode is
// true or deps.VerifyRateLimiter is nil — follows the same nil-disables
// pattern as the other rate limiters on Deps, which keeps tests that don't
// construct a limiter free of verify-endpoint rate-limit state.
func checkVerifyAttemptRateLimit(w http.ResponseWriter, r *http.Request, deps *Deps, agentID int64) bool {
	if deps.DevMode || deps.VerifyRateLimiter == nil {
		return true
	}

	ip := clientIP(r, deps.TrustedProxyHeader)
	if ip == "" {
		return true
	}
	allowed, retryAfter := deps.VerifyRateLimiter.AllowKey("ip:" + ip)
	if allowed {
		return true
	}

	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	log.Printf("[%s] verify rate limited: scope=ip agent_id=%d retry_after=%d",
		TraceID(r.Context()), agentID, retryAfter)
	CaptureMessage(r.Context(), SeverityWarning, fmt.Sprintf(
		"agent verify rate limited (scope=ip, agent_id=%d, retry_after=%ds)",
		agentID, retryAfter,
	))
	RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(
		fmt.Sprintf("Too many verification attempts. Please retry after %d seconds.", retryAfter),
		retryAfter,
	))
	return false
}
