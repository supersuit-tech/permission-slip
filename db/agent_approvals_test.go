package db_test

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

func TestApprovalTTLFromEnv(t *testing.T) {
	silent := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{"Unset", "", 10 * time.Minute},
		{"ValidMinutes", "30m", 30 * time.Minute},
		{"ValidHours", "2h", 2 * time.Hour},
		{"AtMin", "1m", time.Minute},
		{"AtMax", "24h", 24 * time.Hour},
		{"Invalid", "junk", 10 * time.Minute},
		{"BelowMin", "30s", 10 * time.Minute},
		{"AboveMax", "25h", 10 * time.Minute},
		{"Negative", "-5m", 10 * time.Minute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("APPROVAL_REQUEST_TTL", tc.env)
			if got := db.ApprovalTTLFromEnv(silent); got != tc.want {
				t.Errorf("env=%q: got %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestApprovalTTLFromEnv_NilLoggerSafe(t *testing.T) {
	t.Setenv("APPROVAL_REQUEST_TTL", "junk")
	if got := db.ApprovalTTLFromEnv(nil); got != 10*time.Minute {
		t.Errorf("nil logger: got %v, want 10m", got)
	}
}
