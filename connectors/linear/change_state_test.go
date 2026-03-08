package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestChangeState_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueUpdate": map[string]any{
					"success": true,
					"issue": map[string]any{
						"id":         "issue-1",
						"identifier": "ENG-1",
						"state":      map[string]string{"id": "state-2", "name": "Done"},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &changeStateAction{conn: conn}

	params, _ := json.Marshal(changeStateParams{IssueID: "issue-1", StateID: "state-2"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.change_state",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["state_name"] != "Done" {
		t.Errorf("state_name = %q, want Done", data["state_name"])
	}
}

func TestChangeState_MissingIssueID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &changeStateAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"state_id": "state-2"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.change_state",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing issue_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestChangeState_MissingStateID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &changeStateAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"issue_id": "issue-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.change_state",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing state_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestChangeState_SuccessFalse(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueUpdate": map[string]any{
					"success": false,
					"issue":   nil,
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &changeStateAction{conn: conn}

	params, _ := json.Marshal(changeStateParams{IssueID: "issue-1", StateID: "state-2"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.change_state",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for success=false")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
