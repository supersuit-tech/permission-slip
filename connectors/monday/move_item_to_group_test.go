package monday

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMoveItemToGroup_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"move_item_to_group": map[string]any{
					"id":   "12345",
					"name": "Moved Task",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &moveItemToGroupAction{conn: conn}

	params, _ := json.Marshal(moveItemToGroupParams{
		ItemID:  "12345",
		GroupID: "done",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.move_item_to_group",
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
	if data["name"] != "Moved Task" {
		t.Errorf("expected name 'Moved Task', got %q", data["name"])
	}
}

func TestMoveItemToGroup_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &moveItemToGroupAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"group_id": "done",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.move_item_to_group",
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

func TestMoveItemToGroup_MissingGroupID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &moveItemToGroupAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"item_id": "12345",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.move_item_to_group",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing group_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMoveItemToGroup_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &moveItemToGroupAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.move_item_to_group",
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
