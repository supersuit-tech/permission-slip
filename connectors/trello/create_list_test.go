package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateList_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/lists" {
			t.Errorf("path = %s, want /lists", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["idBoard"] != testBoardID {
			t.Errorf("idBoard = %v, want %s", reqBody["idBoard"], testBoardID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":      testListID,
			"name":    "Backlog",
			"idBoard": testBoardID,
			"closed":  false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_list"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_list",
		Parameters:  json.RawMessage(`{"board_id":"` + testBoardID + `","name":"Backlog"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["name"] != "Backlog" {
		t.Errorf("name = %v, want Backlog", data["name"])
	}
}

func TestCreateList_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_list"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_list",
		Parameters:  json.RawMessage(`{"board_id":"` + testBoardID + `"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
