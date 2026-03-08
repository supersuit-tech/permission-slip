package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListCycles_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"cycles": map[string]any{
					"nodes": []map[string]any{
						{
							"id":        "cycle-1",
							"name":      "Cycle 1",
							"number":    1,
							"startsAt":  "2024-01-01T00:00:00Z",
							"endsAt":    "2024-01-15T00:00:00Z",
							"completedAt": nil,
						},
						{
							"id":          "cycle-2",
							"name":        "Cycle 2",
							"number":      2,
							"startsAt":    "2024-01-15T00:00:00Z",
							"endsAt":      "2024-01-29T00:00:00Z",
							"completedAt": "2024-01-29T00:00:00Z",
						},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &listCyclesAction{conn: conn}

	params, _ := json.Marshal(listCyclesParams{TeamID: "team-1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.list_cycles",
		Parameters:  params,
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

func TestListCycles_MissingTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCyclesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.list_cycles",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing team_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListCycles_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCyclesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.list_cycles",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
