package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListAccounts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/query") {
			t.Errorf("path = %s, expected to contain /query", r.URL.Path)
		}
		// Verify query parameter contains Account query.
		if !strings.Contains(r.URL.RawQuery, "Account") {
			t.Errorf("query = %s, expected to contain Account", r.URL.RawQuery)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{
				"Account": []map[string]any{
					{"Id": "1", "Name": "Checking", "AccountType": "Bank"},
					{"Id": "2", "Name": "Savings", "AccountType": "Bank"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_accounts"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_accounts",
		Parameters:  json.RawMessage(`{"account_type":"Bank"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(data))
	}
}

func TestListAccounts_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_accounts"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_accounts",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty array, got %d items", len(data))
	}
}

func TestListAccounts_NoFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if strings.Contains(query, "WHERE") {
			t.Errorf("query should not contain WHERE when no filter: %s", query)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"QueryResponse": map[string]any{
				"Account": []map[string]any{
					{"Id": "1", "Name": "Checking"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.list_accounts"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_accounts",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListAccounts_InvalidAccountType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.list_accounts"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.list_accounts",
		Parameters:  json.RawMessage(`{"account_type":"'; DROP TABLE Account; --"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid account type, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
