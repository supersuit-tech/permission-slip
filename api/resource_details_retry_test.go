package api

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRetryResourceDetails_SucceedsAfterTransientFailures(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	want := map[string]any{"subject": "Hello"}

	details, err := retryResourceDetails(context.Background(), func(context.Context) (map[string]any, error) {
		n := calls.Add(1)
		if n < 3 {
			return nil, &connectors.ExternalError{Message: "connection reset"}
		}
		return want, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["subject"] != want["subject"] {
		t.Fatalf("details = %v, want %v", details, want)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls = %d, want 3", calls.Load())
	}
}

func TestRetryResourceDetails_AuthErrorNoRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	_, err := retryResourceDetails(context.Background(), func(context.Context) (map[string]any, error) {
		calls.Add(1)
		return nil, &connectors.AuthError{Message: "bad token"}
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryResourceDetails_ValidationErrorNoRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	_, err := retryResourceDetails(context.Background(), func(context.Context) (map[string]any, error) {
		calls.Add(1)
		return nil, &connectors.ValidationError{Message: "bad params"}
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryResourceDetails_CanceledErrorNoRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	_, err := retryResourceDetails(context.Background(), func(context.Context) (map[string]any, error) {
		calls.Add(1)
		return nil, &connectors.CanceledError{Message: "canceled"}
	})
	if err == nil {
		t.Fatal("expected canceled error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryResourceDetails_UnknownErrorNoRetry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	_, err := retryResourceDetails(context.Background(), func(context.Context) (map[string]any, error) {
		calls.Add(1)
		return nil, errors.New("something weird")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestRetryResourceDetails_RespectsRateLimitRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	start := time.Now()
	_, err := retryResourceDetails(context.Background(), func(context.Context) (map[string]any, error) {
		n := calls.Add(1)
		if n == 1 {
			return nil, &connectors.RateLimitError{Message: "slow down", RetryAfter: 50 * time.Millisecond}
		}
		return map[string]any{"ok": true}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
	if elapsed := time.Since(start); elapsed < 45*time.Millisecond {
		t.Fatalf("elapsed = %v, expected at least ~50ms backoff", elapsed)
	}
}

func TestRetryResourceDetails_StopsWhenContextDeadlineExhausted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	var calls atomic.Int32
	_, err := retryResourceDetails(ctx, func(context.Context) (map[string]any, error) {
		calls.Add(1)
		return nil, &connectors.TimeoutError{Message: "slow"}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsTimeoutError(err) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if calls.Load() < 1 {
		t.Fatalf("expected at least one attempt, got %d", calls.Load())
	}
}
