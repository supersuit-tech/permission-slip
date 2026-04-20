package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateSpreadsheet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/spreadsheets" {
			t.Errorf("expected path /spreadsheets, got %s", r.URL.Path)
		}

		var body sheetsCreateSpreadsheetRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties.Title != "Q1 Report" {
			t.Errorf("expected title 'Q1 Report', got %q", body.Properties.Title)
		}
		if len(body.Sheets) != 0 {
			t.Errorf("expected no sheets in request, got %d", len(body.Sheets))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsCreateSpreadsheetResponse{
			SpreadsheetID:  "spreadsheet-abc-123",
			SpreadsheetURL: "https://docs.google.com/spreadsheets/d/spreadsheet-abc-123/edit",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{Title: "Q1 Report"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["spreadsheet_id"] != "spreadsheet-abc-123" {
		t.Errorf("expected spreadsheet_id 'spreadsheet-abc-123', got %q", data["spreadsheet_id"])
	}
	if data["spreadsheet_url"] != "https://docs.google.com/spreadsheets/d/spreadsheet-abc-123/edit" {
		t.Errorf("unexpected spreadsheet_url: %q", data["spreadsheet_url"])
	}
	if data["title"] != "Q1 Report" {
		t.Errorf("expected title 'Q1 Report', got %q", data["title"])
	}
}

func TestCreateSpreadsheet_WithSheetTitles(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body sheetsCreateSpreadsheetRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Sheets) != 2 {
			t.Fatalf("expected 2 sheets, got %d", len(body.Sheets))
		}
		if body.Sheets[0].Properties.Title != "Jan" {
			t.Errorf("expected first sheet 'Jan', got %q", body.Sheets[0].Properties.Title)
		}
		if body.Sheets[1].Properties.Title != "Feb" {
			t.Errorf("expected second sheet 'Feb', got %q", body.Sheets[1].Properties.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsCreateSpreadsheetResponse{
			SpreadsheetID: "spreadsheet-multi-456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Title:       "Budget",
		SheetTitles: []string{"Jan", "Feb"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["spreadsheet_id"] != "spreadsheet-multi-456" {
		t.Errorf("expected spreadsheet_id 'spreadsheet-multi-456', got %q", data["spreadsheet_id"])
	}
	// URL should be auto-generated when the API doesn't return one.
	if data["spreadsheet_url"] != "https://docs.google.com/spreadsheets/d/spreadsheet-multi-456/edit" {
		t.Errorf("unexpected spreadsheet_url: %q", data["spreadsheet_url"])
	}
}

func TestCreateSpreadsheet_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{Title: "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_spreadsheet",
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

func TestCreateSpreadsheet_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_spreadsheet",
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
