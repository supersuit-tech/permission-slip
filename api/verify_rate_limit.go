package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// verifyAttemptLimiter is a singleton, lazily-initialized RateLimiter dedicated
// to confirmation-code verification attempts. It enforces:
//
//   - per-agent_id: 1 attempt/sec sustained, burst of 5.
//     Aligned with maxVerificationAttempts so callers using the correct code
//     under normal conditions are unaffected, but anyone trying to brute-force
//     another agent's code is throttled to a useless rate.
//   - per-IP: 0.5 attempts/sec sustained, burst of 10.
//     Stops a single IP from probing many agent IDs in quick succession.
//
// The buckets are per-process (in-memory). For multi-instance deployments the
// effective limit scales with instance count, which is acceptable as a defense
// in depth on top of the underlying ~50-bit confirmation code entropy.
var (
	verifyByAgentLimiter *RateLimiter
	verifyByIPLimiter    *RateLimiter
	verifyLimiterOnce    sync.Once
)

func initVerifyLimiters() {
	verifyByAgentLimiter = NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  1,
		PerKeyBurst: 5,
		GlobalRate:  10000, // global bucket unused; pre-auth middleware enforces it.
		GlobalBurst: 10000,
	})
	verifyByIPLimiter = NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  0.5,
		PerKeyBurst: 10,
		GlobalRate:  10000,
		GlobalBurst: 10000,
	})
}

// checkVerifyAttemptRateLimit enforces tight per-agent and per-IP throttling
// on POST /agents/{agent_id}/verify so that confirmation code brute-force
// attempts are infeasible regardless of how many agents the caller probes.
//
// Returns true when the request is allowed; otherwise it writes a 429 response
// with a Retry-After header and returns false. In dev mode this is a no-op.
func checkVerifyAttemptRateLimit(w http.ResponseWriter, r *http.Request, deps *Deps, agentID int64) bool {
	if deps.DevMode {
		return true
	}
	verifyLimiterOnce.Do(initVerifyLimiters)

	agentKey := fmt.Sprintf("agent:%d", agentID)
	if allowed, retryAfter := verifyByAgentLimiter.AllowKey(agentKey); !allowed {
		respondVerifyRateLimited(w, r, agentID, "agent", retryAfter)
		return false
	}

	ip := clientIP(r, deps.TrustedProxyHeader)
	if ip != "" {
		if allowed, retryAfter := verifyByIPLimiter.AllowKey("ip:" + ip); !allowed {
			respondVerifyRateLimited(w, r, agentID, "ip", retryAfter)
			return false
		}
	}
	return true
}

func respondVerifyRateLimited(w http.ResponseWriter, r *http.Request, agentID int64, scope string, retryAfter int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	log.Printf("[%s] verify rate limited: scope=%s agent_id=%d retry_after=%d",
		TraceID(r.Context()), scope, agentID, retryAfter)
	CaptureMessage(r.Context(), SeverityWarning, fmt.Sprintf(
		"agent verify rate limited (scope=%s, agent_id=%d, retry_after=%ds)",
		scope, agentID, retryAfter,
	))
	RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(
		fmt.Sprintf("Too many verification attempts. Please retry after %d seconds.", retryAfter),
		retryAfter,
	))
}

// resetVerifyLimitersForTesting clears the verification rate-limiter buckets.
// Tests that hit the verify endpoint repeatedly can call this between cases
// so a previous test's attempts don't bleed into the next one.
func resetVerifyLimitersForTesting() {
	verifyLimiterOnce = sync.Once{}
	verifyByAgentLimiter = nil
	verifyByIPLimiter = nil
}
