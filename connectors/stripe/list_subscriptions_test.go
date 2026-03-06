package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
		if got := r.URL.Query().Get("customer"); got != "cus_123" {
			t.Errorf("customer = %q, want %q", got, "cus_123")
		}
		if got := r.URL.Query().Get("status"); got != "active" {
			t.Errorf("status = %q, want %q", got, "active")
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want %q", got, "5")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "sub_1", "status": "active"},
				{"id": "sub_2", "status": "active"},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_subscriptions"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{"customer_id":"cus_123","status":"active","limit":5}`),
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
		t.Fatalf("data field is not an array: %T", data["data"])
	}
	if len(subs) != 2 {
		t.Errorf("len(data) = %d, want 2", len(subs))
	}
	if data["has_more"] != false {
		t.Errorf("has_more = %v, want false", data["has_more"])
	}
}

func TestListSubscriptions_DefaultLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want %q (default)", got, "10")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data":     []any{},
			"has_more": false,
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
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListSubscriptions_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.list_subscriptions"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_subscriptions",
		Parameters:  json.RawMessage(`{bad`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListSubscriptions_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "authentication_error",
				"message": "Invalid API Key",
			},
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
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
