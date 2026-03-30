package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetProfitLoss_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/reports/ProfitAndLoss") {
			t.Errorf("path = %s, expected to contain /reports/ProfitAndLoss", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Header": map[string]any{
				"ReportName": "ProfitAndLoss",
			},
			"Rows": map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.get_profit_loss"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.get_profit_loss",
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
	if header["ReportName"] != "ProfitAndLoss" {
		t.Errorf("ReportName = %v, want ProfitAndLoss", header["ReportName"])
	}
}

func TestGetProfitLoss_NoDateParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not have query params when no dates provided.
		if strings.Contains(r.URL.RawQuery, "start_date") {
			t.Errorf("unexpected start_date in query: %s", r.URL.RawQuery)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Header": map[string]any{"ReportName": "ProfitAndLoss"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.get_profit_loss"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.get_profit_loss",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
