package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListBoards_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/members/me/boards" {
			t.Errorf("path = %s, want /members/me/boards", r.URL.Path)
		}
		if r.URL.Query().Get("filter") != "open" {
			t.Errorf("filter = %q, want open", r.URL.Query().Get("filter"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": testBoardID, "name": "My Board", "closed": false, "shortUrl": "https://trello.com/b/abc"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.list_boards"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.list_boards",
		Parameters:  json.RawMessage(`{}`),
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
		t.Errorf("expected 1 board, got %d", len(data))
	}
	if data[0]["id"] != testBoardID {
		t.Errorf("id = %v, want %s", data[0]["id"], testBoardID)
	}
}
