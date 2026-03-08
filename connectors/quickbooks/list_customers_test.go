package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListQBCustomers_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "SELECT * FROM Customer WHERE Active = true") {
			t.Errorf("query = %q, missing Active = true filter", query)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{
				"Customer": []map[string]any{
					{"Id": "1", "DisplayName": "Alice Corp"},
					{"Id": "2", "DisplayName": "Bob LLC"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_customers"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_customers",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(data) != 2 {
		t.Errorf("got %d customers, want 2", len(data))
	}
}

func TestListQBCustomers_WithDisplayNameFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "DisplayName LIKE '%Alice%'") {
			t.Errorf("query missing DisplayName filter, got: %q", query)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{
				"Customer": []map[string]any{
					{"Id": "1", "DisplayName": "Alice Corp"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_customers"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_customers",
		Parameters:  json.RawMessage(`{"display_name": "Alice"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListQBCustomers_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_customers"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_customers",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty slice, got %d items", len(data))
	}
}

func TestListQBCustomers_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.list_customers"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_customers",
		Parameters:  json.RawMessage(`{bad}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
