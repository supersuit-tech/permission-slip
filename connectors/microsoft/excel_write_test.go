package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestExcelWriteRange_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		var req graphWriteRangeRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if len(req.Values) != 2 {
			t.Errorf("expected 2 rows in request, got %d", len(req.Values))
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
	action := &excelWriteRangeAction{conn: conn}

	params, _ := json.Marshal(excelWriteRangeParams{
		ItemID:    "item-123",
		SheetName: "Sheet1",
		Range:     "A1:B2",
		Values: [][]any{
			{"Name", "Age"},
			{"Alice", 30},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_write_range",
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
}

func TestExcelWriteRange_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelWriteRangeAction{conn: conn}

	params, _ := json.Marshal(excelWriteRangeParams{
		SheetName: "Sheet1",
		Range:     "A1:B2",
		Values:    [][]any{{"a"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_write_range",
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

func TestExcelWriteRange_MissingValues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelWriteRangeAction{conn: conn}

	params, _ := json.Marshal(excelWriteRangeParams{
		ItemID:    "item-123",
		SheetName: "Sheet1",
		Range:     "A1:B2",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_write_range",
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

func TestExcelWriteRange_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelWriteRangeAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_write_range",
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
