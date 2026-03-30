package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateIssue_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueCreate": map[string]any{
					"success": true,
					"issue": map[string]any{
						"id":         "issue-uuid-1",
						"identifier": "ENG-123",
						"title":      "Fix login bug",
						"url":        "https://linear.app/team/issue/ENG-123",
						"state":      map[string]string{"name": "Backlog"},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createIssueAction{conn: conn}

	params, _ := json.Marshal(createIssueParams{
		TeamID: "team-1",
		Title:  "Fix login bug",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
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
	if data["id"] != "issue-uuid-1" {
		t.Errorf("id = %q, want %q", data["id"], "issue-uuid-1")
	}
	if data["identifier"] != "ENG-123" {
		t.Errorf("identifier = %q, want %q", data["identifier"], "ENG-123")
	}
	if data["title"] != "Fix login bug" {
		t.Errorf("title = %q, want %q", data["title"], "Fix login bug")
	}
	if data["url"] != "https://linear.app/team/issue/ENG-123" {
		t.Errorf("url = %q, want %q", data["url"], "https://linear.app/team/issue/ENG-123")
	}
	if data["state"] != "Backlog" {
		t.Errorf("state = %q, want %q", data["state"], "Backlog")
	}
}

func TestCreateIssue_WithAllOptionalFields(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueCreate": map[string]any{
					"success": true,
					"issue": map[string]any{
						"id":         "issue-uuid-2",
						"identifier": "ENG-456",
						"title":      "Add feature X",
						"url":        "https://linear.app/team/issue/ENG-456",
						"state":      map[string]string{"name": "Todo"},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createIssueAction{conn: conn}

	priority := 2
	params, _ := json.Marshal(createIssueParams{
		TeamID:      "team-1",
		Title:       "Add feature X",
		Description: "Detailed description",
		AssigneeID:  "user-1",
		Priority:    &priority,
		StateID:     "state-1",
		LabelIDs:    []string{"label-1", "label-2"},
		ProjectID:   "project-1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
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
	if data["identifier"] != "ENG-456" {
		t.Errorf("identifier = %q, want %q", data["identifier"], "ENG-456")
	}
}

func TestCreateIssue_MissingTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createIssueAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"title": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
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

func TestCreateIssue_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createIssueAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"team_id": "team-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateIssue_InvalidPriority(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createIssueAction{conn: conn}

	priority := 5
	params, _ := json.Marshal(createIssueParams{
		TeamID:   "team-1",
		Title:    "Test",
		Priority: &priority,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
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

func TestCreateIssue_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createIssueAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
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

func TestCreateIssue_GraphQLError(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"errors": []map[string]any{
				{
					"message":    "Team not found",
					"extensions": map[string]any{"type": "validation_error"},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createIssueAction{conn: conn}

	params, _ := json.Marshal(createIssueParams{
		TeamID: "invalid-team",
		Title:  "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for GraphQL error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateIssue_SuccessFalse(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueCreate": map[string]any{
					"success": false,
					"issue":   nil,
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &createIssueAction{conn: conn}

	params, _ := json.Marshal(createIssueParams{
		TeamID: "team-1",
		Title:  "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.create_issue",
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
