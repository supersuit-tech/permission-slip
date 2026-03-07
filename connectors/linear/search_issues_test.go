package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchIssues_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{
						{
							"id":         "issue-1",
							"identifier": "ENG-100",
							"title":      "Login bug",
							"priority":   1,
							"url":        "https://linear.app/team/issue/ENG-100",
							"state":      map[string]string{"name": "In Progress"},
							"assignee":   map[string]string{"name": "Alice"},
						},
						{
							"id":         "issue-2",
							"identifier": "ENG-101",
							"title":      "Login redesign",
							"priority":   3,
							"url":        "https://linear.app/team/issue/ENG-101",
							"state":      map[string]string{"name": "Todo"},
							"assignee":   nil,
						},
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &searchIssuesAction{conn: conn}

	params, _ := json.Marshal(searchIssuesParams{
		Query: "login",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.search_issues",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data []searchIssueResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 results, got %d", len(data))
	}
	if data[0].Identifier != "ENG-100" {
		t.Errorf("first result identifier = %q, want %q", data[0].Identifier, "ENG-100")
	}
	if data[0].Assignee != "Alice" {
		t.Errorf("first result assignee = %q, want %q", data[0].Assignee, "Alice")
	}
	if data[0].Priority != "1" {
		t.Errorf("first result priority = %q, want %q", data[0].Priority, "1")
	}
	// Second result has nil assignee.
	if data[1].Assignee != "" {
		t.Errorf("second result assignee = %q, want empty", data[1].Assignee)
	}
}

func TestSearchIssues_WithFilters(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &searchIssuesAction{conn: conn}

	params, _ := json.Marshal(searchIssuesParams{
		Query:      "bug",
		TeamID:     "team-1",
		AssigneeID: "user-1",
		State:      "In Progress",
		Limit:      10,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.search_issues",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data []searchIssueResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected 0 results, got %d", len(data))
	}
}

func TestSearchIssues_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchIssuesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.search_issues",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing query")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchIssues_LimitExceedsMax(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchIssuesAction{conn: conn}

	params, _ := json.Marshal(searchIssuesParams{
		Query: "test",
		Limit: 200,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.search_issues",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for limit > max")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchIssues_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchIssuesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.search_issues",
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

func TestSearchIssues_EmptyResults(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes": []map[string]any{},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &searchIssuesAction{conn: conn}

	params, _ := json.Marshal(searchIssuesParams{Query: "nonexistent"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.search_issues",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data []searchIssueResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty results, got %d", len(data))
	}
}
