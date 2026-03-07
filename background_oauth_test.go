package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestOAuthRefreshInterval(t *testing.T) {
	// Cannot use t.Parallel() because subtests use t.Setenv.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name     string
		envValue string
		want     time.Duration
	}{
		{name: "default when unset", envValue: "", want: 10 * time.Minute},
		{name: "valid duration 5m", envValue: "5m", want: 5 * time.Minute},
		{name: "valid duration 1m (minimum)", envValue: "1m", want: time.Minute},
		{name: "valid duration 30m", envValue: "30m", want: 30 * time.Minute},
		{name: "invalid string falls back", envValue: "notaduration", want: 10 * time.Minute},
		{name: "zero duration falls back", envValue: "0s", want: 10 * time.Minute},
		{name: "negative duration falls back", envValue: "-5m", want: 10 * time.Minute},
		{name: "below minimum falls back", envValue: "30s", want: 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OAUTH_REFRESH_INTERVAL", tt.envValue)
			got := oauthRefreshInterval(logger)
			if got != tt.want {
				t.Errorf("oauthRefreshInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartOAuthRefresh_StopsOnContextCancel(t *testing.T) {
	// Cannot use t.Parallel() because we set OAUTH_REFRESH_INTERVAL via Setenv.
	ctx, cancel := context.WithCancel(context.Background())
	t.Setenv("OAUTH_REFRESH_INTERVAL", "1m")

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Use a nil DB/vault/registry — runOAuthRefresh will fail to list connections
	// but the goroutine lifecycle is what we're testing.
	deps := OAuthRefreshDeps{}

	done := startOAuthRefresh(ctx, deps, logger)

	// Give the goroutine time to run the initial pass.
	time.Sleep(50 * time.Millisecond)

	cancel()
	select {
	case <-done:
		// Success — goroutine exited.
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not exit within 2s after context cancel")
	}

	output := buf.String()
	if !strings.Contains(output, "oauth refresh: scheduled") {
		t.Errorf("expected startup log message, got: %s", output)
	}
}
