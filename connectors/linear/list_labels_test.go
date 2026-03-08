package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListLabels_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueLabels": map[string]any{
					"nodes": []map[string]any{
						{"id": "label-1", "name": "bug", "color": "#FF0000", "isGroup": false},
						{"id": "label-2", "name": "feature", "color": "#00FF00", "isGroup": false},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &listLabelsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.list_labels",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["total_count"] != float64(2) {
		t.Errorf("total_count = %v, want 2", data["total_count"])
	}
}

func TestListLabels_WithTeamFilter(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueLabels": map[string]any{
					"nodes": []map[string]any{
						{"id": "label-1", "name": "bug", "color": "#FF0000", "isGroup": false},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &listLabelsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.list_labels",
		Parameters:  json.RawMessage(`{"team_id":"team-1"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["total_count"] != float64(1) {
		t.Errorf("total_count = %v, want 1", data["total_count"])
	}
}
