package agentwake

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	maxAttempts        = 5
	initialBackoff     = 2 * time.Second
	maxBackoff         = 32 * time.Second
)

// HTTPDoer performs HTTP requests. *http.Client implements this interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DeliverPOST sends delivery with up to maxAttempts tries and exponential backoff.
// Returns nil when any attempt receives a 2xx response.
func DeliverPOST(ctx context.Context, client HTTPDoer, delivery *Delivery) error {
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	backoff := initialBackoff
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := deliverOnce(ctx, client, delivery)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == maxAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
	return lastErr
}

func deliverOnce(ctx context.Context, client HTTPDoer, delivery *Delivery) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.URL, bytes.NewReader(delivery.Body))
	if err != nil {
		return err
	}
	for k, v := range delivery.Headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return &httpStatusError{code: resp.StatusCode}
}

type httpStatusError struct {
	code int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("webhook returned HTTP %d", e.code)
}

// DeliverAsync fires delivery in a goroutine with retries; outcomes are logged.
func DeliverAsync(logger *slog.Logger, client HTTPDoer, delivery *Delivery, logAttrs ...any) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		err := DeliverPOST(ctx, client, delivery)
		attrs := append([]any{"url", delivery.URL}, logAttrs...)
		if err != nil {
			attrs = append(attrs, "error", err)
			if logger != nil {
				logger.Warn("agent wake webhook delivery failed", attrs...)
			}
			return
		}
		if logger != nil {
			logger.Info("agent wake webhook delivered", attrs...)
		}
	}()
}
