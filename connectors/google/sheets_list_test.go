package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSheetsListSheets_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/spreadsheets/spreadsheet-789" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("fields") != "sheets.properties" {
			t.Errorf("expected fields=sheets.properties, got %s", r.URL.Query().Get("fields"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsSpreadsheetResponse{
			Sheets: []struct {
				Properties sheetsProperties `json:"properties"`
			}{
				{Properties: sheetsProperties{SheetID: 0, Title: "Sheet1", Index: 0, SheetType: "GRID"}},
				{Properties: sheetsProperties{SheetID: 123, Title: "Data", Index: 1, SheetType: "GRID"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsListSheetsAction{conn: conn}

	params, _ := json.Marshal(sheetsListSheetsParams{SpreadsheetID: "spreadsheet-789"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_list_sheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Sheets []sheetSummary `json:"sheets"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Sheets) != 2 {
		t.Fatalf("expected 2 sheets, got %d", len(data.Sheets))
	}
	if data.Sheets[0].Title != "Sheet1" {
		t.Errorf("expected first sheet title 'Sheet1', got %q", data.Sheets[0].Title)
	}
	if data.Sheets[1].SheetID != 123 {
		t.Errorf("expected second sheet ID 123, got %d", data.Sheets[1].SheetID)
	}
}

func TestSheetsListSheets_EmptySpreadsheet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsSpreadsheetResponse{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsListSheetsAction{conn: conn}

	params, _ := json.Marshal(sheetsListSheetsParams{SpreadsheetID: "spreadsheet-789"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_list_sheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Sheets []sheetSummary `json:"sheets"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Sheets) != 0 {
		t.Errorf("expected 0 sheets, got %d", len(data.Sheets))
	}
}

func TestSheetsListSheets_MissingSpreadsheetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsListSheetsAction{conn: conn}

	params, _ := json.Marshal(sheetsListSheetsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_list_sheets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing spreadsheet_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsListSheets_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsListSheetsAction{conn: conn}

	params, _ := json.Marshal(sheetsListSheetsParams{SpreadsheetID: "abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_list_sheets",
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

func TestSheetsListSheets_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsListSheetsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_list_sheets",
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
