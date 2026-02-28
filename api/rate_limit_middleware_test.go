package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestRateLimitMiddleware_AllowsNormalTraffic(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 5,
		GlobalRate:  100,
		GlobalBurst: 50,
	})

	handler := RateLimitMiddleware(limiter, false, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_Returns429OnPerIPLimit(t *testing.T) {
	t.Parallel()
	now := time.Now()
	limiter := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 1,
		GlobalRate:  100,
		GlobalBurst: 100,
	})
	limiter.nowFunc = func() time.Time { return now }

	handler := RateLimitMiddleware(limiter, false, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request from the same IP should succeed.
	req1 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rec1.Code)
	}

	// Second request from same IP should be rate limited (burst=1).
	req2 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rec2.Code)
	}

	// Check Retry-After header.
	retryAfterStr := rec2.Header().Get("Retry-After")
	if retryAfterStr == "" {
		t.Fatal("missing Retry-After header")
	}
	retryAfter, err := strconv.Atoi(retryAfterStr)
	if err != nil {
		t.Fatalf("invalid Retry-After header: %q", retryAfterStr)
	}
	if retryAfter < 1 {
		t.Fatalf("Retry-After should be at least 1, got %d", retryAfter)
	}

	// Check response body matches spec format.
	var errResp ErrorResponse
	if err := json.NewDecoder(rec2.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp.Error.Code != ErrRateLimited {
		t.Fatalf("expected error code %q, got %q", ErrRateLimited, errResp.Error.Code)
	}
	if !errResp.Error.Retryable {
		t.Fatal("expected retryable=true")
	}
	if errResp.Error.RetryAfter < 1 {
		t.Fatalf("expected retry_after >= 1, got %d", errResp.Error.RetryAfter)
	}
}

func TestRateLimitMiddleware_DevModePassthrough(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 1,
		GlobalRate:  10,
		GlobalBurst: 1,
	})

	handler := RateLimitMiddleware(limiter, true, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow unlimited requests in dev mode.
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("dev mode request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimitMiddleware_DifferentIPsHaveSeparateBuckets(t *testing.T) {
	t.Parallel()
	now := time.Now()
	limiter := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 1,
		GlobalRate:  100,
		GlobalBurst: 100,
	})
	limiter.nowFunc = func() time.Time { return now }

	handler := RateLimitMiddleware(limiter, false, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust burst for 10.0.0.1.
	req1 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req1.RemoteAddr = "10.0.0.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first IP first request: expected 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req2.RemoteAddr = "10.0.0.1:12345"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("first IP second request: expected 429, got %d", rec2.Code)
	}

	// 10.0.0.2 should still be allowed (separate bucket).
	req3 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req3.RemoteAddr = "10.0.0.2:54321"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("second IP first request: expected 200, got %d", rec3.Code)
	}
}

func TestRateLimitMiddleware_GlobalLimitAcrossIPs(t *testing.T) {
	t.Parallel()
	now := time.Now()
	limiter := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  100,
		PerKeyBurst: 100,
		GlobalRate:  10,
		GlobalBurst: 3,
	})
	limiter.nowFunc = func() time.Time { return now }

	handler := RateLimitMiddleware(limiter, false, "")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust global limit across different IPs.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/v1/agents", nil)
		req.RemoteAddr = "10.0.0." + strconv.Itoa(i+1) + ":1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// Next request from a new IP should hit global limit.
	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req.RemoteAddr = "10.0.0.99:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after global burst, got %d", rec.Code)
	}
}

// ── rateLimitKey / clientIP (IP extraction) ─────────────────────────────────

func TestRateLimitKey_ExtractsIPFromRemoteAddr(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"

	key := rateLimitKey(req, "")
	if key != "ip:192.168.1.100" {
		t.Fatalf("expected key %q, got %q", "ip:192.168.1.100", key)
	}
}

func TestRateLimitKey_HandlesIPv6(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:8080"

	key := rateLimitKey(req, "")
	// net.SplitHostPort correctly strips brackets from IPv6 addresses.
	if key != "ip:::1" {
		t.Fatalf("expected key %q, got %q", "ip:::1", key)
	}
}

func TestClientIP_FlyClientIPHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:12345" // proxy IP
	req.Header.Set("Fly-Client-IP", "203.0.113.50")

	ip := clientIP(req, "Fly-Client-IP")
	if ip != "203.0.113.50" {
		t.Fatalf("expected %q, got %q", "203.0.113.50", ip)
	}
}

func TestClientIP_XForwardedForSingleIP(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.10")

	ip := clientIP(req, "X-Forwarded-For")
	if ip != "198.51.100.10" {
		t.Fatalf("expected %q, got %q", "198.51.100.10", ip)
	}
}

func TestClientIP_XForwardedForMultipleIPs(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

	// Should use the leftmost (first) IP.
	ip := clientIP(req, "X-Forwarded-For")
	if ip != "203.0.113.50" {
		t.Fatalf("expected %q, got %q", "203.0.113.50", ip)
	}
}

func TestClientIP_XForwardedForWithSpaces(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	req.Header.Set("X-Forwarded-For", "  203.0.113.50  , 70.41.3.18")

	ip := clientIP(req, "X-Forwarded-For")
	if ip != "203.0.113.50" {
		t.Fatalf("expected %q, got %q", "203.0.113.50", ip)
	}
}

func TestClientIP_FallsBackToRemoteAddrWhenHeaderMissing(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	// No Fly-Client-IP header set.

	ip := clientIP(req, "Fly-Client-IP")
	if ip != "192.168.1.100" {
		t.Fatalf("expected %q, got %q", "192.168.1.100", ip)
	}
}

func TestClientIP_FallsBackToRemoteAddrWhenHeaderEmpty(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	req.Header.Set("Fly-Client-IP", "")

	ip := clientIP(req, "Fly-Client-IP")
	if ip != "192.168.1.100" {
		t.Fatalf("expected %q, got %q", "192.168.1.100", ip)
	}
}

func TestClientIP_FallsBackToRemoteAddrWhenNoProxyConfigured(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:54321"
	req.Header.Set("Fly-Client-IP", "203.0.113.50") // should be ignored

	ip := clientIP(req, "")
	if ip != "192.168.1.100" {
		t.Fatalf("expected %q (RemoteAddr), got %q", "192.168.1.100", ip)
	}
}

func TestClientIP_IPv6InProxyHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	req.Header.Set("Fly-Client-IP", "2001:db8::1")

	ip := clientIP(req, "Fly-Client-IP")
	if ip != "2001:db8::1" {
		t.Fatalf("expected %q, got %q", "2001:db8::1", ip)
	}
}

func TestClientIP_XForwardedForWithIPv6(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "172.16.0.1:12345"
	req.Header.Set("X-Forwarded-For", "2001:db8::1, 198.51.100.10")

	ip := clientIP(req, "X-Forwarded-For")
	if ip != "2001:db8::1" {
		t.Fatalf("expected %q, got %q", "2001:db8::1", ip)
	}
}

func TestClientIP_RemoteAddrWithoutPort(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100" // no port

	ip := clientIP(req, "")
	if ip != "192.168.1.100" {
		t.Fatalf("expected %q, got %q", "192.168.1.100", ip)
	}
}

func TestRateLimitMiddleware_UsesProxyHeaderForRateLimiting(t *testing.T) {
	t.Parallel()
	now := time.Now()
	limiter := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 1,
		GlobalRate:  100,
		GlobalBurst: 100,
	})
	limiter.nowFunc = func() time.Time { return now }

	// Configure middleware with Fly-Client-IP as the trusted header.
	handler := RateLimitMiddleware(limiter, false, "Fly-Client-IP")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Two requests from different RemoteAddrs but the same Fly-Client-IP
	// should share a rate limit bucket.
	req1 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req1.RemoteAddr = "172.16.0.1:11111"
	req1.Header.Set("Fly-Client-IP", "203.0.113.50")
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req2.RemoteAddr = "172.16.0.2:22222" // different proxy IP
	req2.Header.Set("Fly-Client-IP", "203.0.113.50") // same client
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request from same client IP: expected 429, got %d", rec2.Code)
	}

	// A request from a different client IP should still be allowed.
	req3 := httptest.NewRequest("GET", "/api/v1/agents", nil)
	req3.RemoteAddr = "172.16.0.3:33333"
	req3.Header.Set("Fly-Client-IP", "198.51.100.99") // different client
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("request from different client: expected 200, got %d", rec3.Code)
	}
}

// ── checkAgentRateLimit (post-auth) ─────────────────────────────────────────

func TestCheckAgentRateLimit_AllowsWithinBurst(t *testing.T) {
	t.Parallel()
	deps := &Deps{
		AgentRateLimiter: NewRateLimiter(RateLimiterConfig{
			PerKeyRate:  10,
			PerKeyBurst: 2,
			GlobalRate:  10000,
			GlobalBurst: 10000,
		}),
	}

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		if !checkAgentRateLimit(rec, req, deps, 42) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Third should be rejected.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	if checkAgentRateLimit(rec, req, deps, 42) {
		t.Fatal("should be rejected after burst")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header on agent rate limit")
	}
}

func TestCheckAgentRateLimit_IsolatesAgents(t *testing.T) {
	t.Parallel()
	deps := &Deps{
		AgentRateLimiter: NewRateLimiter(RateLimiterConfig{
			PerKeyRate:  10,
			PerKeyBurst: 1,
			GlobalRate:  10000,
			GlobalBurst: 10000,
		}),
	}

	// Exhaust agent 1.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	checkAgentRateLimit(rec, req, deps, 1)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	if checkAgentRateLimit(rec, req, deps, 1) {
		t.Fatal("agent 1 should be rate limited")
	}

	// Agent 2 should still be allowed.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	if !checkAgentRateLimit(rec, req, deps, 2) {
		t.Fatal("agent 2 should not be affected by agent 1")
	}
}

func TestCheckAgentRateLimit_DevModeSkips(t *testing.T) {
	t.Parallel()
	deps := &Deps{
		DevMode: true,
		AgentRateLimiter: NewRateLimiter(RateLimiterConfig{
			PerKeyRate:  1,
			PerKeyBurst: 1,
			GlobalRate:  1,
			GlobalBurst: 1,
		}),
	}

	// Should always allow in dev mode.
	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		if !checkAgentRateLimit(rec, req, deps, 42) {
			t.Fatalf("dev mode request %d should be allowed", i+1)
		}
	}
}

func TestCheckAgentRateLimit_NilLimiterAllows(t *testing.T) {
	t.Parallel()
	deps := &Deps{AgentRateLimiter: nil}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	if !checkAgentRateLimit(rec, req, deps, 42) {
		t.Fatal("nil limiter should allow all requests")
	}
}
