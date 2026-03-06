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
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}
		if got := r.Header.Get("Idempotency-Key"); got == "" {
			t.Error("expected Idempotency-Key header on POST")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("email"); got != "alice@example.com" {
			t.Errorf("email = %q, want alice@example.com", got)
		}
		if got := r.FormValue("name"); got != "Alice Smith" {
			t.Errorf("name = %q, want Alice Smith", got)
		}
		if got := r.FormValue("metadata[tier]"); got != "premium" {
			t.Errorf("metadata[tier] = %q, want premium", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":    "cus_abc123",
			"email": "alice@example.com",
			"name":  "Alice Smith",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"alice@example.com","name":"Alice Smith","metadata":{"tier":"premium"}}`),
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

func TestCreateCustomer_EmailOnly(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("email"); got != "bob@example.com" {
			t.Errorf("email = %q, want bob@example.com", got)
		}
		// Optional fields should not be present.
		if got := r.FormValue("name"); got != "" {
			t.Errorf("name should be empty, got %q", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":    "cus_def456",
			"email": "bob@example.com",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"bob@example.com"}`),
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
		Parameters:  json.RawMessage(`{"name":"No Email"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
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
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Invalid email address",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_customer",
		Parameters:  json.RawMessage(`{"email":"not-valid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
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
		Parameters:  json.RawMessage(`{"email":"alice@example.com"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
