package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchIssues_FullTextSearch(t *testing.T) {
	t.Parallel()

	// When no filters are specified, search uses issueSearch (full-text).
	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"issueSearch": map[string]any{
					"nodes": []map[string]any{
						{
							"id":          "issue-1",
							"identifier":  "ENG-100",
							"title":       "Login bug",
							"description": "Users cannot log in",
							"priority":    1,
							"url":         "https://linear.app/team/issue/ENG-100",
							"state":       map[string]string{"name": "In Progress"},
							"assignee":    map[string]string{"name": "Alice"},
						},
						{
							"id":          "issue-2",
							"identifier":  "ENG-101",
							"title":       "Login redesign",
							"description": "",
							"priority":    3,
							"url":         "https://linear.app/team/issue/ENG-101",
							"state":       map[string]string{"name": "Todo"},
							"assignee":    nil,
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

	var data searchIssuesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalCount != 2 {
		t.Errorf("total_count = %d, want 2", data.TotalCount)
	}
	if len(data.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(data.Issues))
	}
	if data.Issues[0].Identifier != "ENG-100" {
		t.Errorf("first result identifier = %q, want %q", data.Issues[0].Identifier, "ENG-100")
	}
	if data.Issues[0].Assignee != "Alice" {
		t.Errorf("first result assignee = %q, want %q", data.Issues[0].Assignee, "Alice")
	}
	if data.Issues[0].Priority != "1" {
		t.Errorf("first result priority = %q, want %q", data.Issues[0].Priority, "1")
	}
	if data.Issues[0].Description != "Users cannot log in" {
		t.Errorf("first result description = %q, want %q", data.Issues[0].Description, "Users cannot log in")
	}
	// Second result has nil assignee and empty description.
	if data.Issues[1].Assignee != "" {
		t.Errorf("second result assignee = %q, want empty", data.Issues[1].Assignee)
	}
}

func TestSearchIssues_WithFilters(t *testing.T) {
	t.Parallel()

	// When filters are specified, search uses filtered issues endpoint.
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

	var data searchIssuesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalCount != 0 {
		t.Errorf("total_count = %d, want 0", data.TotalCount)
	}
	if len(data.Issues) != 0 {
		t.Errorf("expected 0 results, got %d", len(data.Issues))
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
				"issueSearch": map[string]any{
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

	var data searchIssuesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalCount != 0 {
		t.Errorf("total_count = %d, want 0", data.TotalCount)
	}
	if len(data.Issues) != 0 {
		t.Errorf("expected empty results, got %d", len(data.Issues))
	}
}
