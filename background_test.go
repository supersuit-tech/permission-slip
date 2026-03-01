package main

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestAuditPurgeInterval(t *testing.T) {
	// Cannot use t.Parallel() because subtests use t.Setenv.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name     string
		envValue string
		want     time.Duration
	}{
		{name: "default when unset", envValue: "", want: time.Hour},
		{name: "valid duration 1m (minimum)", envValue: "1m", want: time.Minute},
		{name: "valid duration 30m", envValue: "30m", want: 30 * time.Minute},
		{name: "valid duration 2h", envValue: "2h", want: 2 * time.Hour},
		{name: "valid duration 24h", envValue: "24h", want: 24 * time.Hour},
		{name: "invalid string falls back", envValue: "notaduration", want: time.Hour},
		{name: "zero duration falls back", envValue: "0s", want: time.Hour},
		{name: "negative duration falls back", envValue: "-5m", want: time.Hour},
		{name: "below minimum falls back", envValue: "30s", want: time.Hour},
		{name: "1ms falls back", envValue: "1ms", want: time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUDIT_PURGE_INTERVAL", tt.envValue)
			got := auditPurgeInterval(logger)
			if got != tt.want {
				t.Errorf("auditPurgeInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}
