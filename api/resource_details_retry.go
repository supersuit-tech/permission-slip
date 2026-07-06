package api

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	resourceDetailsMaxAttempts = 3
	resourceDetailsBackoffBase = 500 * time.Millisecond
)

// retryResourceDetails runs the enrichment up to maxAttempts times, backing off
// between tries, retrying only transient failures and stopping early on success,
// a non-retryable error, or context deadline.
func retryResourceDetails(ctx context.Context, fn func(context.Context) (map[string]any, error)) (map[string]any, error) {
	var lastErr error
	for attempt := 1; attempt <= resourceDetailsMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return nil, lastErr
			}
			return nil, err
		}

		details, err := fn(ctx)
		if err == nil {
			return details, nil
		}
		lastErr = err

		if !isRetryableResourceDetailsError(err) {
			return nil, err
		}
		if attempt == resourceDetailsMaxAttempts {
			break
		}

		delay := resourceDetailsRetryDelay(ctx, attempt, err)
		if delay <= 0 {
			break
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, lastErr
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func isRetryableResourceDetailsError(err error) bool {
	if errors.Is(err, context.Canceled) || connectors.IsCanceledError(err) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if connectors.IsAuthError(err) || connectors.IsValidationError(err) {
		return false
	}

	var timeout *connectors.TimeoutError
	var external *connectors.ExternalError
	var rateLimit *connectors.RateLimitError
	switch {
	case errors.As(err, &timeout):
		return true
	case errors.As(err, &external):
		return true
	case errors.As(err, &rateLimit):
		return true
	default:
		log.Printf("retryResourceDetails: non-retryable unclassified error: %v", err)
		return false
	}
}

func resourceDetailsRetryDelay(ctx context.Context, attempt int, err error) time.Duration {
	remaining := time.Duration(1<<63 - 1)
	if deadline, ok := ctx.Deadline(); ok {
		remaining = time.Until(deadline)
		if remaining <= 0 {
			return 0
		}
	}

	var delay time.Duration
	var rateLimit *connectors.RateLimitError
	if errors.As(err, &rateLimit) && rateLimit.RetryAfter > 0 {
		delay = rateLimit.RetryAfter
	} else {
		delay = resourceDetailsBackoffBase
		for i := 1; i < attempt; i++ {
			delay *= 2
		}
		// Jitter ±25% around the base delay.
		jitter := delay / 4
		delay = delay - jitter + time.Duration(rand.Int63n(int64(2*jitter+1)))
	}

	if delay > remaining {
		delay = remaining
	}
	return delay
}
