package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSheetsAppendRows_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/spreadsheets/spreadsheet-456/values/Sheet1:append" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("valueInputOption") != "USER_ENTERED" {
			t.Errorf("expected valueInputOption=USER_ENTERED, got %s", r.URL.Query().Get("valueInputOption"))
		}

		var body sheetsUpdateValuesRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.MajorDimension != "ROWS" {
			t.Errorf("expected majorDimension ROWS, got %s", body.MajorDimension)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsAppendValuesResponse{
			SpreadsheetID: "spreadsheet-456",
			Updates: struct {
				UpdatedRange   string `json:"updatedRange"`
				UpdatedRows    int    `json:"updatedRows"`
				UpdatedColumns int    `json:"updatedColumns"`
				UpdatedCells   int    `json:"updatedCells"`
			}{
				UpdatedRange:   "Sheet1!A4:C5",
				UpdatedRows:    2,
				UpdatedColumns: 3,
				UpdatedCells:   6,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "spreadsheet-456",
		Range:         "Sheet1",
		Values: [][]any{
			{"Charlie", 35, "Chicago"},
			{"Diana", 28, "Seattle"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["updated_range"] != "Sheet1!A4:C5" {
		t.Errorf("expected updated_range 'Sheet1!A4:C5', got %v", data["updated_range"])
	}
	if data["updated_rows"] != float64(2) {
		t.Errorf("expected updated_rows 2, got %v", data["updated_rows"])
	}
}

func TestSheetsAppendRows_ColumnOnlyRangeNormalized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Column-only range "Sheet1!A:C" must be normalized to "Sheet1" before
		// hitting the API — the append endpoint rejects column-only ranges.
		if r.URL.Path != "/spreadsheets/spreadsheet-456/values/Sheet1:append" {
			t.Errorf("expected normalized path .../values/Sheet1:append, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sheetsAppendValuesResponse{
			SpreadsheetID: "spreadsheet-456",
			Updates: struct {
				UpdatedRange   string `json:"updatedRange"`
				UpdatedRows    int    `json:"updatedRows"`
				UpdatedColumns int    `json:"updatedColumns"`
				UpdatedCells   int    `json:"updatedCells"`
			}{UpdatedRange: "Sheet1!A2:C2", UpdatedRows: 1, UpdatedColumns: 3, UpdatedCells: 3},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "spreadsheet-456",
		Range:         "Sheet1!A:C",
		Values:        [][]any{{"Demo Row", "Yes", "Yes"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSheetsAppendRows_MissingSpreadsheetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		Range:  "Sheet1",
		Values: [][]any{{"val"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
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

func TestSheetsAppendRows_MissingValues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing values")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsAppendRows_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 429, "message": "Rate limit exceeded"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1",
		Values:        [][]any{{"val"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestSheetsAppendRows_RaggedRows(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1",
		Values: [][]any{
			{"A", "B"},
			{"C"},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for ragged rows")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsAppendRows_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsAppendRowsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
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
