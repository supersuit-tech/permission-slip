package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestStartAuditPurge_StopsOnContextCancel(t *testing.T) {
	// Cannot use t.Parallel() because we set AUDIT_PURGE_INTERVAL via Setenv.
	ctx, cancel := context.WithCancel(context.Background())
	t.Setenv("AUDIT_PURGE_INTERVAL", "1m")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mock := &purgeDBMock{execResult: pgconn.NewCommandTag("DELETE 0")}

	done := startAuditPurge(ctx, mock, logger)

	// Give the goroutine time to run the initial purge.
	time.Sleep(50 * time.Millisecond)

	// Cancel the context and verify the goroutine exits promptly.
	cancel()
	select {
	case <-done:
		// Success — goroutine exited.
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not exit within 2s after context cancel")
	}
}

func TestRunAuditPurge_LogsInfoOnContextCancel(t *testing.T) {
	t.Parallel()

	// Pre-cancel the context to simulate shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mock := &purgeDBMock{execErr: context.Canceled}

	runAuditPurge(ctx, mock, logger)

	output := buf.String()
	if !strings.Contains(output, "audit purge: cancelled") {
		t.Errorf("expected 'audit purge: cancelled' in log output, got: %s", output)
	}
	if strings.Contains(output, "ERROR") {
		t.Errorf("context cancellation should not log at ERROR level, got: %s", output)
	}
}

func TestRunAuditPurge_LogsErrorOnDBFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mock := &purgeDBMock{execErr: errors.New("connection reset")}

	runAuditPurge(ctx, mock, logger)

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("expected ERROR level log for DB failure, got: %s", output)
	}
	if !strings.Contains(output, "audit purge: failed") {
		t.Errorf("expected 'audit purge: failed' in log output, got: %s", output)
	}
}

func TestRunAuditPurge_DebugLogForZeroRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mock := &purgeDBMock{execResult: pgconn.NewCommandTag("DELETE 0")}

	runAuditPurge(ctx, mock, logger)

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("expected DEBUG level for 0 rows deleted, got: %s", output)
	}
}

func TestRunAuditPurge_InfoLogForDeletedRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	mock := &purgeDBMock{execResult: pgconn.NewCommandTag("DELETE 5")}

	runAuditPurge(ctx, mock, logger)

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected INFO level for rows deleted, got: %s", output)
	}
	// PurgeExpiredAuditEvents calls Exec twice (subscribed + unsubscribed users),
	// so the mock's "DELETE 5" is counted twice → 10 total.
	if !strings.Contains(output, "rows_deleted=10") {
		t.Errorf("expected rows_deleted=10 in output, got: %s", output)
	}
}

// purgeDBMock implements db.DBTX for audit purge testing.
// PurgeExpiredAuditEvents calls Exec twice, so the mock returns the
// configured result/error for each call.
type purgeDBMock struct {
	execResult pgconn.CommandTag
	execErr    error
}

func (m *purgeDBMock) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return m.execResult, m.execErr
}

func (m *purgeDBMock) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *purgeDBMock) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return nil
}
