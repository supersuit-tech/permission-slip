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

func TestGetBalance_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/balance" {
			t.Errorf("path = %s, want /v1/balance", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_abc123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer sk_test_abc123")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"object":   "balance",
			"livemode": false,
			"available": []map[string]any{
				{"amount": 12345, "currency": "usd"},
			},
			"pending": []map[string]any{
				{"amount": 6789, "currency": "usd"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	available, ok := data["available"].([]any)
	if !ok || len(available) != 1 {
		t.Fatalf("expected 1 available balance entry, got %v", data["available"])
	}
	entry := available[0].(map[string]any)
	if entry["amount"] != float64(12345) {
		t.Errorf("amount = %v, want 12345", entry["amount"])
	}
	if entry["currency"] != "usd" {
		t.Errorf("currency = %v, want usd", entry["currency"])
	}
}

func TestGetBalance_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "Invalid API Key provided",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestGetBalance_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
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

func TestGetBalance_NoIdempotencyKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Idempotency-Key"); got != "" {
			t.Errorf("Idempotency-Key should be empty for GET, got %q", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":    "balance",
			"available": []map[string]any{},
			"pending":   []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_balance"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_balance",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
