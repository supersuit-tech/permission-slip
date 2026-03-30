package monday

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListGroups_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var gqlReq map[string]any
		json.Unmarshal(body, &gqlReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"boards": []map[string]any{
					{
						"groups": []map[string]any{
							{"id": "topics", "title": "To Do", "color": "#ff0000"},
							{"id": "done", "title": "Done", "color": "#00ff00"},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["monday.list_groups"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.list_groups",
		Parameters:  json.RawMessage(`{"board_id":"12345"}`),
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
		t.Errorf("expected 2 groups, got %d", len(data))
	}
	if data[0]["title"] != "To Do" {
		t.Errorf("title = %v, want To Do", data[0]["title"])
	}
}

func TestListGroups_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["monday.list_groups"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.list_groups",
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
