package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetIssue_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issue": map[string]any{
					"id":          "issue-uuid-1",
					"identifier":  "ENG-123",
					"title":       "Fix bug",
					"description": "A bug to fix",
					"priority":    2,
					"url":         "https://linear.app/team/issue/ENG-123",
					"state":       map[string]string{"id": "state-1", "name": "In Progress"},
					"assignee":    map[string]string{"id": "user-1", "name": "Alice"},
					"team":        map[string]string{"id": "team-1", "name": "Engineering"},
					"labels":      map[string]any{"nodes": []map[string]string{{"id": "label-1", "name": "bug"}}},
					"createdAt":   "2024-01-01T00:00:00Z",
					"updatedAt":   "2024-01-02T00:00:00Z",
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &getIssueAction{conn: conn}

	params, _ := json.Marshal(getIssueParams{IssueID: "issue-uuid-1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.get_issue",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["identifier"] != "ENG-123" {
		t.Errorf("identifier = %v, want ENG-123", data["identifier"])
	}
}

func TestGetIssue_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getIssueAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.get_issue",
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

func TestGetIssue_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getIssueAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.get_issue",
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
