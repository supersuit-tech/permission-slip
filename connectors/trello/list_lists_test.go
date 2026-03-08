package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListLists_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/"+testBoardID+"/lists" {
			t.Errorf("path = %s, want /boards/%s/lists", r.URL.Path, testBoardID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": testListID, "name": "To Do", "closed": false},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.list_lists"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.list_lists",
		Parameters:  json.RawMessage(`{"board_id":"` + testBoardID + `"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data) != 1 {
		t.Errorf("expected 1 list, got %d", len(data))
	}
	if data[0]["name"] != "To Do" {
		t.Errorf("name = %v, want To Do", data[0]["name"])
	}
}

func TestListLists_InvalidBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.list_lists"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.list_lists",
		Parameters:  json.RawMessage(`{"board_id":"not-a-valid-id"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
