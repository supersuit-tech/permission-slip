package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSheetsReadRange_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/spreadsheets/spreadsheet-123/values/Sheet1!A1:D3" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("unexpected auth header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsValueRange{
			Range:          "Sheet1!A1:D3",
			MajorDimension: "ROWS",
			Values: [][]any{
				{"Name", "Age", "City"},
				{"Alice", 30, "NYC"},
				{"Bob", 25, "LA"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsReadRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsReadRangeParams{
		SpreadsheetID: "spreadsheet-123",
		Range:         "Sheet1!A1:D3",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_read_range",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Range  string  `json:"range"`
		Values [][]any `json:"values"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Range != "Sheet1!A1:D3" {
		t.Errorf("expected range 'Sheet1!A1:D3', got %q", data.Range)
	}
	if len(data.Values) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(data.Values))
	}
}

func TestSheetsReadRange_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsValueRange{
			Range: "Sheet1!A1:D3",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsReadRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsReadRangeParams{
		SpreadsheetID: "spreadsheet-123",
		Range:         "Sheet1!A1:D3",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_read_range",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Values [][]any `json:"values"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Values != nil {
		t.Errorf("expected nil values for empty range, got %v", data.Values)
	}
}

func TestSheetsReadRange_MissingSpreadsheetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsReadRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsReadRangeParams{Range: "Sheet1!A1:D3"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_read_range",
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

func TestSheetsReadRange_MissingRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsReadRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsReadRangeParams{SpreadsheetID: "abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_read_range",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing range")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsReadRange_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsReadRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsReadRangeParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1!A1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_read_range",
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

func TestSheetsReadRange_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsReadRangeAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_read_range",
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
