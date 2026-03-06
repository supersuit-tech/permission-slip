package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestExcelAppendRows_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		expectedPath := "/me/drive/items/item-123/workbook/tables/SalesTable/rows"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var req graphAddRowsRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if len(req.Values) != 2 {
			t.Errorf("expected 2 rows in request, got %d", len(req.Values))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"index": 5,
			"values": [][]any{
				{"Widget", 100, 9.99},
				{"Gadget", 50, 19.99},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &excelAppendRowsAction{conn: conn}

	params, _ := json.Marshal(excelAppendRowsParams{
		ItemID:    "item-123",
		TableName: "SalesTable",
		Values: [][]any{
			{"Widget", 100, 9.99},
			{"Gadget", 50, 19.99},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var ar appendRowsResult
	if err := json.Unmarshal(result.Data, &ar); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if ar.Index != 5 {
		t.Errorf("expected index 5, got %d", ar.Index)
	}
	if len(ar.Values) != 2 {
		t.Errorf("expected 2 rows, got %d", len(ar.Values))
	}
}

func TestExcelAppendRows_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelAppendRowsAction{conn: conn}

	params, _ := json.Marshal(excelAppendRowsParams{
		TableName: "SalesTable",
		Values:    [][]any{{"a"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_append_rows",
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

func TestExcelAppendRows_MissingTableName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelAppendRowsAction{conn: conn}

	params, _ := json.Marshal(excelAppendRowsParams{
		ItemID: "item-123",
		Values: [][]any{{"a"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_append_rows",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing table_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExcelAppendRows_MissingValues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelAppendRowsAction{conn: conn}

	params, _ := json.Marshal(excelAppendRowsParams{
		ItemID:    "item-123",
		TableName: "SalesTable",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_append_rows",
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

func TestExcelAppendRows_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &excelAppendRowsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_append_rows",
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

func TestExcelAppendRows_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "ErrorAccessDenied",
				"message": "Access denied.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &excelAppendRowsAction{conn: conn}

	params, _ := json.Marshal(excelAppendRowsParams{
		ItemID:    "item-123",
		TableName: "SalesTable",
		Values:    [][]any{{"a"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.excel_append_rows",
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
