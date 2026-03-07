package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateIssue_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueUpdate": map[string]any{
					"success": true,
					"issue": map[string]any{
						"id":         "issue-uuid-1",
						"identifier": "ENG-123",
						"title":      "Updated title",
						"url":        "https://linear.app/team/issue/ENG-123",
						"state":      map[string]string{"name": "In Progress"},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &updateIssueAction{conn: conn}

	params, _ := json.Marshal(updateIssueParams{
		IssueID: "issue-uuid-1",
		Title:   "Updated title",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.update_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["title"] != "Updated title" {
		t.Errorf("title = %q, want %q", data["title"], "Updated title")
	}
	if data["state"] != "In Progress" {
		t.Errorf("state = %q, want %q", data["state"], "In Progress")
	}
}

func TestUpdateIssue_MissingIssueID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateIssueAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"title": "New title"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.update_issue",
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

func TestUpdateIssue_InvalidPriority(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateIssueAction{conn: conn}

	priority := -1
	params, _ := json.Marshal(updateIssueParams{
		IssueID:  "issue-1",
		Priority: &priority,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.update_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateIssue_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateIssueAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.update_issue",
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

func TestUpdateIssue_SuccessFalse(t *testing.T) {
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
	action := &updateIssueAction{conn: conn}

	params, _ := json.Marshal(updateIssueParams{
		IssueID: "issue-1",
		Title:   "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.update_issue",
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
