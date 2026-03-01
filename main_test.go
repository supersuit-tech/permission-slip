package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const secondDuration = time.Second

// newLocalListener returns a TCP listener on a random free port.
func newLocalListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

// ── Health endpoint tests ──────────────────────────────────────────────────

func TestHandleHealth_NoDB(t *testing.T) {
	t.Parallel()

	handler := handleHealth(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

func TestHandleHealth_DBHealthy(t *testing.T) {
	t.Parallel()

	handler := handleHealth(&mockDBTX{healthy: true})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
	if resp["database"] != "ok" {
		t.Errorf("expected database 'ok', got %q", resp["database"])
	}
}

func TestHandleHealth_DBUnreachable(t *testing.T) {
	t.Parallel()

	handler := handleHealth(&mockDBTX{healthy: false})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "degraded" {
		t.Errorf("expected status 'degraded', got %q", resp["status"])
	}
	if resp["database"] != "unreachable" {
		t.Errorf("expected database 'unreachable', got %q", resp["database"])
	}
}

// ── Graceful shutdown test ─────────────────────────────────────────────────

func TestGracefulShutdown_DrainsRequests(t *testing.T) {
	t.Parallel()

	// Create a handler that simulates work with a short delay, then responds.
	started := make(chan struct{})
	finish := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started) // signal that the handler is running
		<-finish       // wait until we're told to complete
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{Handler: handler}
	ln, err := newLocalListener()
	if err != nil {
		t.Fatal(err)
	}

	go srv.Serve(ln)

	// Send a request that will be in-flight when we shut down.
	resultCh := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/test")
		if err != nil {
			resultCh <- 0
			return
		}
		resp.Body.Close()
		resultCh <- resp.StatusCode
	}()

	// Wait for the handler to start processing, with a timeout to prevent
	// deadlock if the HTTP GET fails before reaching the handler.
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	// Start graceful shutdown in a goroutine.
	shutdownDone := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*secondDuration)
	defer cancel()
	go func() {
		shutdownDone <- srv.Shutdown(ctx)
	}()

	// Give the server a moment to stop accepting new connections.
	time.Sleep(50 * time.Millisecond)

	// Let the in-flight request complete.
	close(finish)

	// Shutdown should complete without error (it waited for the request).
	if err := <-shutdownDone; err != nil {
		t.Fatalf("shutdown error: %v", err)
	}

	// The in-flight request should have completed with 200.
	status := <-resultCh
	if status != http.StatusOK {
		t.Errorf("expected in-flight request to complete with 200, got %d", status)
	}
}

// ── Audit purge interval tests ─────────────────────────────────────────────

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

// ── Helpers ────────────────────────────────────────────────────────────────

// mockDBTX implements db.DBTX for health check testing.
type mockDBTX struct {
	healthy bool
}

func (m *mockDBTX) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *mockDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockDBTX) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return &mockRow{healthy: m.healthy}
}

// mockRow implements pgx.Row.
type mockRow struct {
	healthy bool
}

func (mr *mockRow) Scan(dest ...any) error {
	if !mr.healthy {
		return errors.New("connection refused")
	}
	if len(dest) > 0 {
		if p, ok := dest[0].(*int); ok {
			*p = 1
		}
	}
	return nil
}
