package monday

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListBoards_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var gqlReq struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		json.Unmarshal(body, &gqlReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []map[string]any{
					{"id": "111", "name": "Board A", "state": "active", "board_kind": "public"},
					{"id": "222", "name": "Board B", "state": "active", "board_kind": "private"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["monday.list_boards"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.list_boards",
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
	if len(data) != 2 {
		t.Errorf("expected 2 boards, got %d", len(data))
	}
	if data[0]["name"] != "Board A" {
		t.Errorf("expected name=Board A, got %v", data[0]["name"])
	}
}

func TestListBoards_InvalidKind(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.list_boards"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.list_boards",
		Parameters:  json.RawMessage(`{"kind":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
