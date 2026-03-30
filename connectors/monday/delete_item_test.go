package monday

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteItem_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var gqlReq map[string]any
		json.Unmarshal(body, &gqlReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"delete_item": map[string]any{
					"id": "12345678",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["monday.delete_item"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.delete_item",
		Parameters:  json.RawMessage(`{"item_id":"12345678"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "deleted" {
		t.Errorf("status = %v, want deleted", data["status"])
	}
	if data["item_id"] != "12345678" {
		t.Errorf("item_id = %v, want 12345678", data["item_id"])
	}
}

func TestDeleteItem_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.delete_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.delete_item",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDeleteItem_InvalidItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.delete_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.delete_item",
		Parameters:  json.RawMessage(`{"item_id":"not-numeric"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
