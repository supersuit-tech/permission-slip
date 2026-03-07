package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSheetsWriteRange_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/spreadsheets/spreadsheet-123/values/Sheet1!A1:C2" {
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
		json.NewEncoder(w).Encode(sheetsUpdateValuesResponse{
			SpreadsheetID:  "spreadsheet-123",
			UpdatedRange:   "Sheet1!A1:C2",
			UpdatedRows:    2,
			UpdatedColumns: 3,
			UpdatedCells:   6,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsWriteRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsWriteRangeParams{
		SpreadsheetID: "spreadsheet-123",
		Range:         "Sheet1!A1:C2",
		Values: [][]any{
			{"Name", "Age", "City"},
			{"Alice", 30, "NYC"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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
	if data["updated_range"] != "Sheet1!A1:C2" {
		t.Errorf("expected updated_range 'Sheet1!A1:C2', got %v", data["updated_range"])
	}
	if data["updated_cells"] != float64(6) {
		t.Errorf("expected updated_cells 6, got %v", data["updated_cells"])
	}
}

func TestSheetsWriteRange_MissingSpreadsheetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsWriteRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsWriteRangeParams{
		Range:  "Sheet1!A1",
		Values: [][]any{{"val"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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

func TestSheetsWriteRange_MissingRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsWriteRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsWriteRangeParams{
		SpreadsheetID: "abc",
		Values:        [][]any{{"val"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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

func TestSheetsWriteRange_MissingValues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsWriteRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsWriteRangeParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1!A1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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

func TestSheetsWriteRange_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "Insufficient Permission"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsWriteRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsWriteRangeParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1!A1",
		Values:        [][]any{{"val"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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

func TestSheetsWriteRange_RaggedRows(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsWriteRangeAction{conn: conn}

	params, _ := json.Marshal(sheetsWriteRangeParams{
		SpreadsheetID: "abc",
		Range:         "Sheet1!A1",
		Values: [][]any{
			{"A", "B", "C"},
			{"D", "E"}, // different length
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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

func TestSheetsWriteRange_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsWriteRangeAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_write_range",
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
