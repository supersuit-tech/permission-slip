package monday

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateItem_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		// Verify column_values is stringified.
		cv, ok := body.Variables["column_values"]
		if !ok {
			t.Error("expected column_values in variables")
		}
		if _, ok := cv.(string); !ok {
			t.Errorf("expected column_values to be a string, got %T", cv)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"change_multiple_column_values": map[string]any{
					"id":   "12345",
					"name": "Updated Task",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateItemAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"board_id": "9876",
		"item_id":  "12345",
		"column_values": map[string]any{
			"status": map[string]string{"label": "Done"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.update_item",
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
	if data["id"] != "12345" {
		t.Errorf("expected id '12345', got %q", data["id"])
	}
}

func TestUpdateItem_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateItemAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"item_id":       "12345",
		"column_values": map[string]string{"status": "Done"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.update_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateItem_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateItemAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"board_id":      "9876",
		"column_values": map[string]string{"status": "Done"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.update_item",
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

func TestUpdateItem_MissingColumnValues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateItemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"board_id": "9876",
		"item_id":  "12345",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.update_item",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing column_values")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateItem_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateItemAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.update_item",
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
