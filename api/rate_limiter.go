package api

import (
	"math"
	"sync"
	"time"
)

// RateLimiter provides in-memory, token-bucket-based rate limiting with
// per-key and global buckets. Keys are typically agent IDs for per-agent
// limiting, but any string key works.
//
// The implementation is safe for concurrent use and periodically evicts
// stale entries to bound memory usage.
type RateLimiter struct {
	mu sync.Mutex

	// Per-key limits.
	perKeyRate  float64       // tokens added per second
	perKeyBurst int           // max tokens (bucket capacity)
	buckets     map[string]*bucket

	// Global limit (single shared bucket).
	global *bucket

	// Cleanup configuration.
	staleAfter time.Duration // evict entries unused for this duration
	lastClean  time.Time

	// nowFunc is used for testing; defaults to time.Now.
	nowFunc func() time.Time
}

// bucket is a single token bucket.
type bucket struct {
	tokens   float64
	lastFill time.Time
	rate     float64 // tokens per second
	burst    int     // max tokens
}

// RateLimiterConfig configures a RateLimiter.
type RateLimiterConfig struct {
	// PerKeyRate is the sustained request rate per key (requests/second).
	PerKeyRate float64
	// PerKeyBurst is the maximum burst size per key.
	PerKeyBurst int

	// GlobalRate is the sustained request rate across all keys (requests/second).
	GlobalRate float64
	// GlobalBurst is the maximum burst size for the global bucket.
	GlobalBurst int
}

// NewRateLimiter creates a new in-memory rate limiter.
func NewRateLimiter(cfg RateLimiterConfig) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		perKeyRate:  cfg.PerKeyRate,
		perKeyBurst: cfg.PerKeyBurst,
		buckets:     make(map[string]*bucket),
		global: &bucket{
			tokens:   float64(cfg.GlobalBurst),
			lastFill: now,
			rate:     cfg.GlobalRate,
			burst:    cfg.GlobalBurst,
		},
		staleAfter: 10 * time.Minute,
		lastClean:  now,
		nowFunc:    time.Now,
	}
}

// Allow checks whether a request identified by key should be allowed.
// It returns (allowed, retryAfterSeconds). When allowed is false,
// retryAfterSeconds is the estimated time until a token becomes available
// (from whichever bucket — per-key or global — is most constrained).
func (rl *RateLimiter) Allow(key string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFunc()
	rl.maybeClean(now)

	// Check global bucket first.
	if !rl.global.allow(now) {
		retryAfter := rl.global.retryAfter(now)
		return false, retryAfter
	}

	// Check per-key bucket (skip if key is empty — e.g., unauthenticated).
	if key != "" {
		b, ok := rl.buckets[key]
		if !ok {
			b = &bucket{
				tokens:   float64(rl.perKeyBurst),
				lastFill: now,
				rate:     rl.perKeyRate,
				burst:    rl.perKeyBurst,
			}
			rl.buckets[key] = b
		}
		if !b.allow(now) {
			// Refund the global token since we're rejecting.
			rl.global.refund()
			retryAfter := b.retryAfter(now)
			return false, retryAfter
		}
	}

	return true, 0
}

// AllowKey checks only the per-key bucket for the given key, without touching
// the global bucket. Use this for post-authentication rate limiting where the
// caller's identity is verified and the global limit is already enforced by
// an earlier middleware layer.
func (rl *RateLimiter) AllowKey(key string) (bool, int) {
	if key == "" {
		return true, 0
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFunc()
	rl.maybeClean(now)

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{
			tokens:   float64(rl.perKeyBurst),
			lastFill: now,
			rate:     rl.perKeyRate,
			burst:    rl.perKeyBurst,
		}
		rl.buckets[key] = b
	}
	if !b.allow(now) {
		retryAfter := b.retryAfter(now)
		return false, retryAfter
	}
	return true, 0
}

// allow attempts to consume one token from the bucket.
// Returns true if a token was available.
func (b *bucket) allow(now time.Time) bool {
	b.refill(now)
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// refund adds one token back (used when a global token was consumed but
// the per-key bucket rejected the request).
func (b *bucket) refund() {
	if b.tokens < float64(b.burst) {
		b.tokens++
	}
}

// refill adds tokens based on elapsed time since last fill.
func (b *bucket) refill(now time.Time) {
	elapsed := now.Sub(b.lastFill).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens += elapsed * b.rate
	if b.tokens > float64(b.burst) {
		b.tokens = float64(b.burst)
	}
	b.lastFill = now
}

// retryAfter returns the number of seconds until the next token is available.
func (b *bucket) retryAfter(now time.Time) int {
	b.refill(now)
	if b.rate <= 0 {
		return 1
	}
	deficit := 1.0 - b.tokens
	if deficit <= 0 {
		return 1 // at least 1 second
	}
	seconds := deficit / b.rate
	retryAfter := int(math.Ceil(seconds))
	if retryAfter < 1 {
		retryAfter = 1
	}
	return retryAfter
}

// maybeClean periodically evicts stale per-key buckets to bound memory.
// Must be called with rl.mu held.
func (rl *RateLimiter) maybeClean(now time.Time) {
	if now.Sub(rl.lastClean) < rl.staleAfter/2 {
		return
	}
	cutoff := now.Add(-rl.staleAfter)
	for key, b := range rl.buckets {
		if b.lastFill.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
	rl.lastClean = now
}
