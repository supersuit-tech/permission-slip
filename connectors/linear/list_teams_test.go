package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListTeams_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"teams": map[string]any{
					"nodes": []map[string]string{
						{"id": "team-1", "name": "Engineering", "key": "ENG", "description": "Engineering team"},
						{"id": "team-2", "name": "Product", "key": "PRD", "description": "Product team"},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &listTeamsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.list_teams",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if data["total_count"] != float64(2) {
		t.Errorf("total_count = %v, want 2", data["total_count"])
	}
}
