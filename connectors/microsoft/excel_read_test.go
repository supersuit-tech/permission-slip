package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestExcelReadRange_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"address": "Sheet1!A1:B2",
			"values": [][]any{
				{"Name", "Age"},
				{"Alice", 30.0},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &excelReadRangeAction{conn: conn}

	params, _ := json.Marshal(excelReadRangeParams{
		ItemID:    "item-123",
		SheetName: "Sheet1",
		Range:     "A1:B2",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_read_range",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rr rangeResult
	if err := json.Unmarshal(result.Data, &rr); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if rr.Address != "Sheet1!A1:B2" {
		t.Errorf("expected address 'Sheet1!A1:B2', got %q", rr.Address)
	}
	if len(rr.Values) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rr.Values))
	}
}

func TestExcelReadRange_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelReadRangeAction{conn: conn}

	params, _ := json.Marshal(excelReadRangeParams{SheetName: "Sheet1", Range: "A1:B2"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_read_range",
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

func TestExcelReadRange_MissingSheetName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelReadRangeAction{conn: conn}

	params, _ := json.Marshal(excelReadRangeParams{ItemID: "item-123", Range: "A1:B2"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_read_range",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing sheet_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExcelReadRange_MissingRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelReadRangeAction{conn: conn}

	params, _ := json.Marshal(excelReadRangeParams{ItemID: "item-123", SheetName: "Sheet1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_read_range",
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

func TestExcelReadRange_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelReadRangeAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_read_range",
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

func TestExcelReadRange_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &excelReadRangeAction{conn: conn}

	params, _ := json.Marshal(excelReadRangeParams{
		ItemID:    "item-123",
		SheetName: "Sheet1",
		Range:     "A1:B2",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_read_range",
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
