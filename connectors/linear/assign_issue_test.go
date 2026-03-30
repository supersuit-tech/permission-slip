package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAssignIssue_Success(t *testing.T) {
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
						"assignee":   map[string]string{"id": "user-1", "name": "Alice"},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &assignIssueAction{conn: conn}

	params, _ := json.Marshal(assignIssueParams{IssueID: "issue-1", AssigneeID: "user-1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.assign_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["assignee_id"] != "user-1" {
		t.Errorf("assignee_id = %q, want user-1", data["assignee_id"])
	}
}

func TestAssignIssue_Unassign(t *testing.T) {
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
						"assignee":   nil,
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &assignIssueAction{conn: conn}

	params, _ := json.Marshal(assignIssueParams{IssueID: "issue-1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.assign_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["id"] != "issue-1" {
		t.Errorf("id = %q, want issue-1", data["id"])
	}
}

func TestAssignIssue_MissingIssueID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignIssueAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"assignee_id": "user-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.assign_issue",
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

func TestAssignIssue_SuccessFalse(t *testing.T) {
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
	action := &assignIssueAction{conn: conn}

	params, _ := json.Marshal(assignIssueParams{IssueID: "issue-1", AssigneeID: "user-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.assign_issue",
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
