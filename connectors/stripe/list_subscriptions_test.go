package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListSubscriptions_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/subscriptions" {
			t.Errorf("path = %s, want /v1/subscriptions", r.URL.Path)
		}
		if got := r.URL.Query().Get("customer"); got != "cus_abc123" {
			t.Errorf("customer = %q, want cus_abc123", got)
		}
		if got := r.URL.Query().Get("status"); got != "active" {
			t.Errorf("status = %q, want active", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"has_more": false,
			"data": []map[string]any{
				{"id": "sub_001", "status": "active", "customer": "cus_abc123"},
				{"id": "sub_002", "status": "active", "customer": "cus_abc123"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_subscriptions"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123","status":"active","limit":5}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	subs, ok := data["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", data["data"])
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
	if data["has_more"] != false {
		t.Errorf("has_more = %v, want false", data["has_more"])
	}
}

func TestListSubscriptions_DefaultLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10 (default)", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"has_more": false,
			"data":     []any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListSubscriptions_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{"status":"invalid_status"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListSubscriptions_LimitTooHigh(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{"limit":101}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListSubscriptions_NoFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("customer"); got != "" {
			t.Errorf("customer should not be set, got %q", got)
		}
		if got := r.URL.Query().Get("status"); got != "" {
			t.Errorf("status should not be set, got %q", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"has_more": true,
			"data":     []any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListSubscriptions_NoIdempotencyKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Idempotency-Key"); got != "" {
			t.Errorf("Idempotency-Key should be empty for GET, got %q", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"has_more": false,
			"data":     []any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListSubscriptions_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
