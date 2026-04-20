package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestSheetsAppendRows_InvalidRange_ListsAvailableTabs verifies that when the
// Google Sheets API returns "Unable to parse range" (i.e. the named tab does
// not exist), the connector remaps it into a ValidationError that lists the
// spreadsheet's actual tab titles. This keeps the error out of Sentry and
// gives the agent enough context to self-correct.
func TestSheetsAppendRows_InvalidRange_ListsAvailableTabs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/spreadsheets/spreadsheet-999/values/Sheet1!A:C:append":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": 400, "message": "Unable to parse range: Sheet1!A:C"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/spreadsheets/spreadsheet-999":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sheetsSpreadsheetResponse{
				Sheets: []struct {
					Properties sheetsProperties `json:"properties"`
				}{
					{Properties: sheetsProperties{Title: "Data"}},
					{Properties: sheetsProperties{Title: "Summary"}},
				},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "spreadsheet-999",
		Range:         "Sheet1!A:C",
		Values:        [][]any{{"a", "b", "c"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid range")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	msg := err.Error()
	for _, want := range []string{`"Sheet1!A:C"`, `"Data"`, `"Summary"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected error message to contain %s, got: %s", want, msg)
		}
	}
}

// TestSheetsAppendRows_InvalidRange_ListFailureFallsBack verifies that if
// Google's "Unable to parse range" 400 is followed by a failed tab-list
// lookup (e.g. permission issue), the original ExternalError is returned
// unchanged — we never mask a genuine external failure with a misleading
// validation message.
func TestSheetsAppendRows_InvalidRange_ListFailureFallsBack(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": 400, "message": "Unable to parse range: Sheet1!A:C"},
			})
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": 500, "message": "backend error"},
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsAppendRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsAppendRowsParams{
		SpreadsheetID: "spreadsheet-999",
		Range:         "Sheet1!A:C",
		Values:        [][]any{{"a", "b", "c"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if connectors.IsValidationError(err) {
		t.Fatalf("expected original ExternalError to be preserved, got ValidationError: %v", err)
	}
	if !connectors.IsExternalError(err) {
		t.Fatalf("expected ExternalError, got %T: %v", err, err)
	}
}
