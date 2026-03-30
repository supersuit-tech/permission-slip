package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateBoard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/boards" {
			t.Errorf("path = %s, want /boards", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["name"] != "Sprint Board" {
			t.Errorf("name = %v, want Sprint Board", reqBody["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       testBoardID,
			"name":     "Sprint Board",
			"shortUrl": "https://trello.com/b/newboard",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_board"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_board",
		Parameters:  json.RawMessage(`{"name":"Sprint Board"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["name"] != "Sprint Board" {
		t.Errorf("name = %v, want Sprint Board", data["name"])
	}
}

func TestCreateBoard_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_board"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_board",
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
