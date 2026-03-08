package monday

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateBoard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var gqlReq map[string]any
		json.Unmarshal(body, &gqlReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"create_board": map[string]any{
					"id":         "99999",
					"name":       "New Board",
					"board_kind": "public",
					"url":        "https://monday.com/boards/99999",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["monday.create_board"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_board",
		Parameters:  json.RawMessage(`{"name":"New Board"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["name"] != "New Board" {
		t.Errorf("name = %v, want New Board", data["name"])
	}
	if data["id"] != "99999" {
		t.Errorf("id = %v, want 99999", data["id"])
	}
}

func TestCreateBoard_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.create_board"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_board",
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
