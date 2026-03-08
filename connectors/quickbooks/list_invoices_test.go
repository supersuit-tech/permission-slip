package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListInvoices_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v3/company/1234567890/query") {
			t.Errorf("path = %s, want /v3/company/1234567890/query*", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "SELECT * FROM Invoice") {
			t.Errorf("query = %q, want SELECT * FROM Invoice", query)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{
				"Invoice": []map[string]any{
					{"Id": "1001", "Balance": 100.0},
					{"Id": "1002", "Balance": 200.0},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_invoices"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_invoices",
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
		t.Errorf("got %d invoices, want 2", len(data))
	}
}

func TestListInvoices_WithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "CustomerRef = '42'") {
			t.Errorf("query missing CustomerRef filter, got: %q", query)
		}
		if !strings.Contains(query, "TxnDate >= '2025-01-01'") {
			t.Errorf("query missing start_date filter, got: %q", query)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{
				"Invoice": []map[string]any{
					{"Id": "1001"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_invoices"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_invoices",
		Parameters:  json.RawMessage(`{"customer_id": "42", "start_date": "2025-01-01"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListInvoices_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_invoices"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_invoices",
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

func TestListInvoices_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.list_invoices"]

	tests := []struct {
		name   string
		params string
	}{
		{"invalid start_date", `{"start_date": "01/01/2025"}`},
		{"invalid end_date", `{"end_date": "2025/12/31"}`},
		{"negative max_results", `{"max_results": -1}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "quickbooks.list_invoices",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
