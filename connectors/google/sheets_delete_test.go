package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestColumnIndexToA1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		index int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{51, "AZ"},
		{52, "BA"},
	}
	for _, tt := range tests {
		if got := columnIndexToA1(tt.index); got != tt.want {
			t.Errorf("columnIndexToA1(%d) = %q, want %q", tt.index, got, tt.want)
		}
	}
}

func TestQuoteA1SheetName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"Sheet1", "Sheet1"},
		{"My Sheet", "'My Sheet'"},
		{"O'Brien", "'O''Brien'"},
		{"Data_2026", "Data_2026"},
	}
	for _, tt := range tests {
		if got := quoteA1SheetName(tt.in); got != tt.want {
			t.Errorf("quoteA1SheetName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMergeDeleteRanges(t *testing.T) {
	t.Parallel()
	got := mergeDeleteRanges([]sheetsDeleteRange{
		{StartIndex: 20, EndIndex: 22},
		{StartIndex: 10, EndIndex: 12},
		{StartIndex: 12, EndIndex: 14},
	})
	want := []sheetsDeleteRange{
		{StartIndex: 10, EndIndex: 14},
		{StartIndex: 20, EndIndex: 22},
	}
	if len(got) != len(want) {
		t.Fatalf("mergeDeleteRanges len = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("mergeDeleteRanges[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDeleteRangesBottomUp(t *testing.T) {
	t.Parallel()
	got := deleteRangesBottomUp([]sheetsDeleteRange{
		{StartIndex: 2, EndIndex: 3},
		{StartIndex: 10, EndIndex: 12},
	})
	if got[0].StartIndex != 10 || got[1].StartIndex != 2 {
		t.Errorf("expected higher index first, got %+v", got)
	}
}

func TestFormatDimensionLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		dimension string
		ranges    []sheetsDeleteRange
		want      string
	}{
		{sheetsDimensionRows, []sheetsDeleteRange{{10, 12}}, "rows 11–12"},
		{sheetsDimensionRows, []sheetsDeleteRange{{0, 1}}, "rows 1"},
		{sheetsDimensionColumns, []sheetsDeleteRange{{0, 2}}, "columns A–B"},
		{sheetsDimensionRows, []sheetsDeleteRange{{10, 12}, {20, 21}}, "rows 11–12 and 21"},
	}
	for _, tt := range tests {
		if got := formatDimensionLabel(tt.dimension, tt.ranges); got != tt.want {
			t.Errorf("formatDimensionLabel(%q, %+v) = %q, want %q", tt.dimension, tt.ranges, got, tt.want)
		}
	}
}

func TestSheetsDeleteRows_SuccessBySheetID(t *testing.T) {
	t.Parallel()

	var gotBody sheetsBatchUpdateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/spreadsheets/spreadsheet-123":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"properties": map[string]any{"title": "Budget"},
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/spreadsheets/spreadsheet-123:batchUpdate":
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("decode body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sheetsBatchUpdateResponse{SpreadsheetID: "spreadsheet-123"})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsDeleteRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "spreadsheet-123",
		SheetID:       intPtr(0),
		StartIndex:    intPtr(10),
		EndIndex:      intPtr(12),
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(gotBody.Requests) != 1 {
		t.Fatalf("expected 1 deleteDimension request, got %d", len(gotBody.Requests))
	}
	del := gotBody.Requests[0].DeleteDimension
	if del == nil {
		t.Fatal("expected deleteDimension request")
	}
	if del.Range.SheetID != 0 || del.Range.Dimension != "ROWS" || del.Range.StartIndex != 10 || del.Range.EndIndex != 12 {
		t.Errorf("unexpected range: %+v", del.Range)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["sheet_name"] != "Sheet1" {
		t.Errorf("expected sheet_name Sheet1, got %v", data["sheet_name"])
	}
	if data["deleted_count"] != float64(2) {
		t.Errorf("expected deleted_count 2, got %v", data["deleted_count"])
	}
	if data["dimension"] != "ROWS" {
		t.Errorf("expected dimension ROWS, got %v", data["dimension"])
	}
}

func TestSheetsDeleteRows_MultipleRangesBottomUp(t *testing.T) {
	t.Parallel()

	var gotBody sheetsBatchUpdateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"properties": map[string]any{"title": "Budget"},
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 7, "title": "Data"}},
				},
			})
		case r.Method == http.MethodPost:
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sheetsBatchUpdateResponse{SpreadsheetID: "ss-1"})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsDeleteRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "ss-1",
		SheetName:     "Data",
		Ranges: []sheetsDeleteRange{
			{StartIndex: 2, EndIndex: 3},
			{StartIndex: 10, EndIndex: 12},
		},
	})

	if _, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotBody.Requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(gotBody.Requests))
	}
	if gotBody.Requests[0].DeleteDimension.Range.StartIndex != 10 {
		t.Errorf("first delete should be the higher index (10), got %d", gotBody.Requests[0].DeleteDimension.Range.StartIndex)
	}
	if gotBody.Requests[1].DeleteDimension.Range.StartIndex != 2 {
		t.Errorf("second delete should be index 2, got %d", gotBody.Requests[1].DeleteDimension.Range.StartIndex)
	}
	if gotBody.Requests[0].DeleteDimension.Range.SheetID != 7 {
		t.Errorf("expected resolved sheet_id 7, got %d", gotBody.Requests[0].DeleteDimension.Range.SheetID)
	}
}

func TestSheetsDeleteRows_DeleteColumns(t *testing.T) {
	t.Parallel()

	var gotBody sheetsBatchUpdateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"properties": map[string]any{"title": "Budget"},
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				},
			})
		case r.Method == http.MethodPost:
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sheetsBatchUpdateResponse{SpreadsheetID: "ss-1"})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsDeleteRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "ss-1",
		SheetID:       intPtr(0),
		StartIndex:    intPtr(0),
		EndIndex:      intPtr(2),
		Dimension:     "COLUMNS",
	})

	if _, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody.Requests[0].DeleteDimension.Range.Dimension != "COLUMNS" {
		t.Errorf("expected COLUMNS, got %s", gotBody.Requests[0].DeleteDimension.Range.Dimension)
	}
}

func TestSheetsDeleteRows_UnknownSheetName(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected only the tab lookup GET, got %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"properties": map[string]any{"title": "Budget"},
			"sheets": []map[string]any{
				{"properties": map[string]any{"sheetId": 0, "title": "Dashboard"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsDeleteRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "ss-1",
		SheetName:     "Sheet1",
		StartIndex:    intPtr(1),
		EndIndex:      intPtr(2),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for unknown sheet name")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), `"Dashboard"`) {
		t.Errorf("expected available tabs in error, got: %s", err.Error())
	}
}

func TestSheetsDeleteRows_MissingSpreadsheetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsDeleteRowsAction{conn: conn}
	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SheetID:    intPtr(0),
		StartIndex: intPtr(0),
		EndIndex:   intPtr(1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
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

func TestSheetsDeleteRows_MissingSheetIdentity(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsDeleteRowsAction{conn: conn}
	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "abc",
		StartIndex:    intPtr(0),
		EndIndex:      intPtr(1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing sheet_id/sheet_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsDeleteRows_InvalidRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsDeleteRowsAction{conn: conn}
	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "abc",
		SheetID:       intPtr(0),
		StartIndex:    intPtr(5),
		EndIndex:      intPtr(5),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for end_index == start_index")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsDeleteRows_InvalidDimension(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsDeleteRowsAction{conn: conn}
	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "abc",
		SheetID:       intPtr(0),
		StartIndex:    intPtr(0),
		EndIndex:      intPtr(1),
		Dimension:     "CELLS",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid dimension")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSheetsDeleteRows_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sheetsDeleteRowsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
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

func TestSheetsDeleteRows_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"properties": map[string]any{"title": "Budget"},
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "Insufficient Permission"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", "", srv.URL)
	action := &sheetsDeleteRowsAction{conn: conn}

	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "abc",
		SheetID:       intPtr(0),
		StartIndex:    intPtr(0),
		EndIndex:      intPtr(1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.sheets_delete_rows",
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

func TestSheetsDeleteRows_ValidateParams(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.ValidateParams("google.sheets_delete_rows", json.RawMessage(`{"spreadsheet_id":"abc"}`))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}

	params, _ := json.Marshal(sheetsDeleteRowsParams{
		SpreadsheetID: "abc",
		SheetID:       intPtr(0),
		StartIndex:    intPtr(1),
		EndIndex:      intPtr(3),
	})
	if err := conn.ValidateParams("google.sheets_delete_rows", params); err != nil {
		t.Fatalf("expected valid params, got %v", err)
	}
}
