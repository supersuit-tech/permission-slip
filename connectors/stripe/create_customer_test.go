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

func TestCreateCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/customers" {
			t.Errorf("path = %s, want /v1/customers", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_abc123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer sk_test_abc123")
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want %q", got, "application/x-www-form-urlencoded")
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("expected Idempotency-Key header to be set")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("email"); got != "alice@example.com" {
			t.Errorf("email = %q, want %q", got, "alice@example.com")
		}
		if got := r.FormValue("name"); got != "Alice" {
			t.Errorf("name = %q, want %q", got, "Alice")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "cus_abc123",
			"email": "alice@example.com",
			"name":  "Alice",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"alice@example.com","name":"Alice"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "cus_abc123" {
		t.Errorf("id = %v, want cus_abc123", data["id"])
	}
	if data["email"] != "alice@example.com" {
		t.Errorf("email = %v, want alice@example.com", data["email"])
	}
}

func TestCreateCustomer_WithMetadata(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("metadata[order_id]"); got != "12345" {
			t.Errorf("metadata[order_id] = %q, want %q", got, "12345")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    "cus_meta",
			"email": "bob@example.com",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"bob@example.com","metadata":{"order_id":"12345"}}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateCustomer_MissingEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"name":"Alice"}`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_AuthError(t *testing.T) {
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
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"test@example.com"}`),
		Credentials: validCreds(),
	})
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "rate_limit_error",
				"message": "Too many requests",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"test@example.com"}`),
		Credentials: validCreds(),
	})
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"test@example.com"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
