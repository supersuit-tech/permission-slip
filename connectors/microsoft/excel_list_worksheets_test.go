package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestExcelListWorksheets_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		expectedPath := "/me/drive/items/item-123/workbook/worksheets"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":         "ws-1",
					"name":       "Sheet1",
					"position":   0,
					"visibility": "Visible",
				},
				{
					"id":         "ws-2",
					"name":       "Sheet2",
					"position":   1,
					"visibility": "Visible",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &excelListWorksheetsAction{conn: conn}

	params, _ := json.Marshal(excelListWorksheetsParams{ItemID: "item-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_list_worksheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []worksheetSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 worksheets, got %d", len(summaries))
	}
	if summaries[0].Name != "Sheet1" {
		t.Errorf("expected name 'Sheet1', got %q", summaries[0].Name)
	}
	if summaries[1].Name != "Sheet2" {
		t.Errorf("expected name 'Sheet2', got %q", summaries[1].Name)
	}
}

func TestExcelListWorksheets_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelListWorksheetsAction{conn: conn}

	params, _ := json.Marshal(excelListWorksheetsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_list_worksheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExcelListWorksheets_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelListWorksheetsAction{conn: conn}

	params, _ := json.Marshal(excelListWorksheetsParams{ItemID: "../secret"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_list_worksheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExcelListWorksheets_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelListWorksheetsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_list_worksheets",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExcelListWorksheets_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token is invalid.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &excelListWorksheetsAction{conn: conn}

	params, _ := json.Marshal(excelListWorksheetsParams{ItemID: "item-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_list_worksheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
