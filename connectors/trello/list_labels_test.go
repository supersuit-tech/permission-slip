package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListLabels_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/boards/"+testBoardID+"/labels" {
			t.Errorf("path = %s, want /boards/%s/labels", r.URL.Path, testBoardID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "507f1f77bcf86cd799439055", "name": "Bug", "color": "red", "idBoard": testBoardID},
			{"id": "507f1f77bcf86cd799439066", "name": "Feature", "color": "green", "idBoard": testBoardID},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.list_labels"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.list_labels",
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
	if len(data) != 2 {
		t.Errorf("expected 2 labels, got %d", len(data))
	}
}
