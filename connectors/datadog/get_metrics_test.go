package datadog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetMetrics_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("path = %s, want /api/v1/query", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "avg:system.cpu.user{*}" {
			t.Errorf("query = %q, want %q", got, "avg:system.cpu.user{*}")
		}
		if got := r.Header.Get("DD-API-KEY"); got != "dd_test_api_key_123" {
			t.Errorf("DD-API-KEY = %q, want %q", got, "dd_test_api_key_123")
		}
		if got := r.Header.Get("DD-APPLICATION-KEY"); got != "dd_test_app_key_456" {
			t.Errorf("DD-APPLICATION-KEY = %q, want %q", got, "dd_test_app_key_456")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"series": []map[string]any{
				{"metric": "system.cpu.user", "pointlist": [][]float64{{1700000000, 42.5}}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_metrics"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_metrics",
		Parameters:  json.RawMessage(`{"query":"avg:system.cpu.user{*}","from":1700000000,"to":1700003600}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestGetMetrics_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.get_metrics"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing query", params: `{"from":1700000000,"to":1700003600}`},
		{name: "missing from", params: `{"query":"avg:system.cpu.user{*}","to":1700003600}`},
		{name: "missing to", params: `{"query":"avg:system.cpu.user{*}","from":1700000000}`},
		{name: "from >= to", params: `{"query":"avg:system.cpu.user{*}","from":1700003600,"to":1700000000}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "datadog.get_metrics",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestGetMetrics_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []string{"Forbidden"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_metrics"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_metrics",
		Parameters:  json.RawMessage(`{"query":"avg:system.cpu.user{*}","from":1700000000,"to":1700003600}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestGetMetrics_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_metrics"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "datadog.get_metrics",
		Parameters:  json.RawMessage(`{"query":"avg:system.cpu.user{*}","from":1700000000,"to":1700003600}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestGetMetrics_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []string{"Rate limit exceeded"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_metrics"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_metrics",
		Parameters:  json.RawMessage(`{"query":"avg:system.cpu.user{*}","from":1700000000,"to":1700003600}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", rle.RetryAfter)
		}
	}
}

func TestGetMetrics_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []string{"Internal Server Error"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_metrics"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_metrics",
		Parameters:  json.RawMessage(`{"query":"avg:system.cpu.user{*}","from":1700000000,"to":1700003600}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
