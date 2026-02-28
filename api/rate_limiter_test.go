package api

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_AllowWithinBurst(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 5,
		GlobalRate:  100,
		GlobalBurst: 50,
	})

	// Should allow up to burst size.
	for i := 0; i < 5; i++ {
		ok, _ := rl.Allow("agent:1")
		if !ok {
			t.Fatalf("request %d should be allowed (within burst)", i+1)
		}
	}

	// 6th request should be rejected.
	ok, retryAfter := rl.Allow("agent:1")
	if ok {
		t.Fatal("request exceeding burst should be rejected")
	}
	if retryAfter < 1 {
		t.Fatalf("retryAfter should be at least 1, got %d", retryAfter)
	}
}

func TestRateLimiter_PerKeyIsolation(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 2,
		GlobalRate:  100,
		GlobalBurst: 100,
	})

	// Exhaust agent:1's burst.
	for i := 0; i < 2; i++ {
		ok, _ := rl.Allow("agent:1")
		if !ok {
			t.Fatalf("agent:1 request %d should be allowed", i+1)
		}
	}
	ok, _ := rl.Allow("agent:1")
	if ok {
		t.Fatal("agent:1 should be rate limited")
	}

	// agent:2 should still be allowed (separate bucket).
	ok, _ = rl.Allow("agent:2")
	if !ok {
		t.Fatal("agent:2 should not be affected by agent:1's limit")
	}
}

func TestRateLimiter_GlobalLimit(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  100,
		PerKeyBurst: 100,
		GlobalRate:  10,
		GlobalBurst: 3,
	})

	// Exhaust the global burst across different keys.
	for i := 0; i < 3; i++ {
		key := "agent:" + string(rune('a'+i))
		ok, _ := rl.Allow(key)
		if !ok {
			t.Fatalf("request %d should be allowed (global burst not yet exceeded)", i+1)
		}
	}

	// Next request should hit global limit even though per-key has room.
	ok, retryAfter := rl.Allow("agent:d")
	if ok {
		t.Fatal("should be rejected by global rate limit")
	}
	if retryAfter < 1 {
		t.Fatalf("retryAfter should be at least 1, got %d", retryAfter)
	}
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	t.Parallel()
	now := time.Now()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10, // 10 tokens/sec
		PerKeyBurst: 2,
		GlobalRate:  100,
		GlobalBurst: 100,
	})
	rl.nowFunc = func() time.Time { return now }

	// Exhaust burst.
	rl.Allow("agent:1")
	rl.Allow("agent:1")
	ok, _ := rl.Allow("agent:1")
	if ok {
		t.Fatal("should be rate limited after burst")
	}

	// Advance time by 200ms → should refill 2 tokens (10/s * 0.2s = 2).
	now = now.Add(200 * time.Millisecond)
	ok, _ = rl.Allow("agent:1")
	if !ok {
		t.Fatal("should be allowed after token refill")
	}
}

func TestRateLimiter_EmptyKeyOnlyGlobal(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  1,
		PerKeyBurst: 1,
		GlobalRate:  100,
		GlobalBurst: 5,
	})

	// Empty key → only global limit applies. Should allow up to global burst.
	for i := 0; i < 5; i++ {
		ok, _ := rl.Allow("")
		if !ok {
			t.Fatalf("request %d with empty key should be allowed (within global burst)", i+1)
		}
	}
	ok, _ := rl.Allow("")
	if ok {
		t.Fatal("should hit global limit")
	}
}

func TestRateLimiter_GlobalRefundOnPerKeyReject(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 1,
		GlobalRate:  100,
		GlobalBurst: 5,
	})

	// Exhaust per-key burst for agent:1.
	ok, _ := rl.Allow("agent:1")
	if !ok {
		t.Fatal("first request should be allowed")
	}

	// This should be rejected by per-key limit but refund the global token.
	ok, _ = rl.Allow("agent:1")
	if ok {
		t.Fatal("should be rejected by per-key limit")
	}

	// Global should still have 4 tokens available (1 consumed, 1 refunded = 4).
	for i := 0; i < 4; i++ {
		ok, _ = rl.Allow("agent:" + string(rune('a'+i)))
		if !ok {
			t.Fatalf("request %d to different agent should be allowed", i+1)
		}
	}
}

func TestRateLimiter_RetryAfterEstimate(t *testing.T) {
	t.Parallel()
	now := time.Now()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  1, // 1 token/sec
		PerKeyBurst: 1,
		GlobalRate:  100,
		GlobalBurst: 100,
	})
	rl.nowFunc = func() time.Time { return now }

	// Consume the one token.
	rl.Allow("agent:1")

	// Next request should report a 1s retry, matching the per-key rate.
	ok, retryAfter := rl.Allow("agent:1")
	if ok {
		t.Fatal("should be rejected")
	}
	if retryAfter != 1 {
		t.Fatalf("retryAfter should be exactly 1 second, got %d", retryAfter)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	now := time.Now()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  100,
		PerKeyBurst: 10,
		GlobalRate:  1000,
		GlobalBurst: 100,
	})
	// Pin time so no refill occurs — makes the test deterministic.
	rl.nowFunc = func() time.Time { return now }

	var wg sync.WaitGroup
	const goroutines = 20
	const requestsPerGoroutine = 50

	allowed := make([]int, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				ok, _ := rl.Allow("agent:shared")
				if ok {
					allowed[idx]++
				}
			}
		}(i)
	}
	wg.Wait()

	total := 0
	for _, n := range allowed {
		total += n
	}

	// With time pinned, no refill occurs, so exactly the per-key burst (10)
	// should be allowed.
	if total != 10 {
		t.Fatalf("expected exactly 10 allowed requests (burst), got %d", total)
	}
}

func TestRateLimiter_AllowKeySkipsGlobal(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 5,
		GlobalRate:  10,
		GlobalBurst: 1, // very restrictive global
	})

	// AllowKey should not touch the global bucket.
	for i := 0; i < 5; i++ {
		ok, _ := rl.AllowKey("agent:1")
		if !ok {
			t.Fatalf("AllowKey request %d should be allowed (within per-key burst)", i+1)
		}
	}

	// Per-key should be exhausted.
	ok, retryAfter := rl.AllowKey("agent:1")
	if ok {
		t.Fatal("AllowKey should reject when per-key burst is exhausted")
	}
	if retryAfter < 1 {
		t.Fatalf("retryAfter should be at least 1, got %d", retryAfter)
	}

	// Global bucket should still have its 1 token (untouched by AllowKey).
	ok, _ = rl.Allow("other:key")
	if !ok {
		t.Fatal("global bucket should be untouched by AllowKey calls")
	}
}

func TestRateLimiter_AllowKeyEmptyKeyAlwaysAllows(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  1,
		PerKeyBurst: 1,
		GlobalRate:  1,
		GlobalBurst: 1,
	})

	ok, _ := rl.AllowKey("")
	if !ok {
		t.Fatal("AllowKey with empty key should always allow")
	}
}

func TestRateLimiter_StaleEntryCleanup(t *testing.T) {
	t.Parallel()
	now := time.Now()
	rl := NewRateLimiter(RateLimiterConfig{
		PerKeyRate:  10,
		PerKeyBurst: 5,
		GlobalRate:  100,
		GlobalBurst: 100,
	})
	rl.nowFunc = func() time.Time { return now }
	rl.staleAfter = 1 * time.Minute

	// Create some entries.
	rl.Allow("agent:old")
	rl.Allow("agent:new")

	// Advance past stale threshold.
	now = now.Add(2 * time.Minute)

	// Access only agent:new so it gets a fresh lastFill.
	rl.Allow("agent:new")

	// Advance past cleanup interval again.
	now = now.Add(2 * time.Minute)

	// Trigger cleanup via any Allow call.
	rl.Allow("agent:trigger")

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, exists := rl.buckets["agent:old"]; exists {
		t.Fatal("stale entry 'agent:old' should have been evicted")
	}
}
