package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListCustomers_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/customers" {
			t.Errorf("path = %s, want /v1/customers", r.URL.Path)
		}
		if got := r.URL.Query().Get("email"); got != "alice@example.com" {
			t.Errorf("email query = %q, want alice@example.com", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit query = %q, want 10", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"data":     []map[string]any{{"id": "cus_abc123", "email": "alice@example.com"}},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_customers"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_customers",
		Parameters:  json.RawMessage(`{"email":"alice@example.com"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["has_more"] != false {
		t.Errorf("has_more = %v, want false", data["has_more"])
	}
}

func TestListCustomers_NoFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10 (default)", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"data":     []any{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_customers"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_customers",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListCustomers_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.list_customers"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_customers",
		Parameters:  json.RawMessage(`{"limit":200}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for limit > 100, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListCustomers_Pagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("starting_after"); got != "cus_previous" {
			t.Errorf("starting_after = %q, want cus_previous", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"data":     []any{},
			"has_more": false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.list_customers"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.list_customers",
		Parameters:  json.RawMessage(`{"starting_after":"cus_previous"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
