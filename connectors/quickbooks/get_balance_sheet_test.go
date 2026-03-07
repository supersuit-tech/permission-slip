package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetBalanceSheet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/reports/BalanceSheet") {
			t.Errorf("path = %s, expected to contain /reports/BalanceSheet", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Header": map[string]any{
				"ReportName": "BalanceSheet",
			},
			"Rows": map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.get_balance_sheet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.get_balance_sheet",
		Parameters:  json.RawMessage(`{"start_date":"2025-01-01","end_date":"2025-12-31"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	header, ok := data["Header"].(map[string]any)
	if !ok {
		t.Fatal("expected Header in response")
	}
	if header["ReportName"] != "BalanceSheet" {
		t.Errorf("ReportName = %v, want BalanceSheet", header["ReportName"])
	}
}

func TestGetBalanceSheet_NoDateParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"Header": map[string]any{"ReportName": "BalanceSheet"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.get_balance_sheet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.get_balance_sheet",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
