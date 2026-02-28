package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// DefaultRateLimiterConfig returns rate limit settings suitable for production
// pre-auth middleware. Keys are client IPs (more generous than per-agent since
// multiple clients may share an IP via NAT).
//
// Per-IP:  50 req/s sustained, burst of 100.
// Global: 200 req/s sustained, burst of 400.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		PerKeyRate:  50,
		PerKeyBurst: 100,
		GlobalRate:  200,
		GlobalBurst: 400,
	}
}

// DefaultAgentRateLimiterConfig returns rate limit settings for post-auth
// per-agent rate limiting. Used after the agent's identity is cryptographically
// verified. The global bucket is unused (set high) because the pre-auth
// middleware already enforces global limits.
//
// Per-agent: 20 req/s sustained, burst of 40.
func DefaultAgentRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		PerKeyRate:  20,
		PerKeyBurst: 40,
		GlobalRate:  10000,
		GlobalBurst: 10000,
	}
}

// RateLimitMiddleware returns middleware that enforces per-IP and global rate
// limits using an in-memory token bucket. When a limit is exceeded, it returns
// HTTP 429 with the standard error format and Retry-After header.
//
// The key is the client's IP address, extracted from trustedProxyHeader
// (e.g. "Fly-Client-IP") when behind a reverse proxy, falling back to
// RemoteAddr for direct connections or local development.
//
// This runs before authentication, so identity-based (per-agent) rate limiting
// is handled separately after auth verification — see checkAgentRateLimit.
//
// When devMode is true the middleware is a no-op passthrough.
func RateLimitMiddleware(limiter *RateLimiter, devMode bool, trustedProxyHeader string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if devMode || limiter == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r, trustedProxyHeader)

			allowed, retryAfter := limiter.Allow(key)
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				log.Printf("[%s] rate limited: key=%q retry_after=%d", TraceID(r.Context()), key, retryAfter)
				RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(
					fmt.Sprintf("Too many requests. Please retry after %d seconds.", retryAfter),
					retryAfter,
				))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkAgentRateLimit enforces per-agent rate limiting after the agent's
// identity has been cryptographically verified. Returns true if the request
// is allowed; on rejection it writes a 429 response and returns false.
func checkAgentRateLimit(w http.ResponseWriter, r *http.Request, deps *Deps, agentID int64) bool {
	if deps.DevMode || deps.AgentRateLimiter == nil {
		return true
	}
	key := fmt.Sprintf("agent:%d", agentID)
	allowed, retryAfter := deps.AgentRateLimiter.AllowKey(key)
	if !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		log.Printf("[%s] agent rate limited: agent_id=%d retry_after=%d", TraceID(r.Context()), agentID, retryAfter)
		RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(
			fmt.Sprintf("Too many requests. Please retry after %d seconds.", retryAfter),
			retryAfter,
		))
		return false
	}
	return true
}

// rateLimitKey extracts the pre-auth rate-limit key from the request.
// Uses the client IP address from the trusted proxy header when configured,
// falling back to RemoteAddr.
func rateLimitKey(r *http.Request, trustedProxyHeader string) string {
	return "ip:" + clientIP(r, trustedProxyHeader)
}

// clientIP extracts the real client IP address from the request.
//
// When trustedProxyHeader is set (e.g. "Fly-Client-IP"), the IP is read from
// that header. This is necessary when running behind a reverse proxy where
// RemoteAddr is always the proxy's IP. Fly-Client-IP is set by Fly.io's proxy
// and cannot be spoofed by clients, making it the recommended default for
// Fly.io deployments.
//
// If the header contains a comma-separated list (as with X-Forwarded-For),
// the leftmost (first) IP is used. Note: X-Forwarded-For can be spoofed by
// clients prepending arbitrary IPs, so Fly-Client-IP is preferred when
// available.
//
// Falls back to RemoteAddr (with port stripped) when no trusted header is
// configured or the header is absent — suitable for local development or
// direct connections.
func clientIP(r *http.Request, trustedProxyHeader string) string {
	if trustedProxyHeader != "" {
		if val := r.Header.Get(trustedProxyHeader); val != "" {
			// Handle comma-separated lists (e.g. X-Forwarded-For: client, proxy1, proxy2).
			// Take the leftmost entry — the original client IP.
			ip := val
			if i := strings.IndexByte(ip, ','); i != -1 {
				ip = ip[:i]
			}
			ip = strings.TrimSpace(ip)
			if ip != "" {
				return ip
			}
		}
	}

	// Fall back to RemoteAddr when no proxy header is present.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr may lack a port in some contexts; return as-is.
		return r.RemoteAddr
	}
	return host
}
