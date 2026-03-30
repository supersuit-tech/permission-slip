package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/customers/cus_abc123" {
			t.Errorf("path = %s, want /v1/customers/cus_abc123", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":      "cus_abc123",
			"email":   "alice@example.com",
			"name":    "Alice Smith",
			"created": 1709740800,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_customer"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_customer",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123"}`),
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

func TestGetCustomer_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.get_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_customer",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetCustomer_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "No such customer: cus_notfound",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.get_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.get_customer",
		Parameters:  json.RawMessage(`{"customer_id":"cus_notfound"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
}

func TestGetCustomer_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	_, err := conn.Actions()["stripe.get_customer"].Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.get_customer",
		Parameters:  json.RawMessage(`{"customer_id":"cus_abc123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
