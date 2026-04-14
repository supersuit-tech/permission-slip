package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listIssuesMixedResponse returns 3 issues — one of which is a pull request
// (identified by the presence of a "pull_request" key per GitHub's API).
func listIssuesMixedResponse() []map[string]any {
	return []map[string]any{
		{"number": 1, "title": "bug report"},
		{"number": 2, "title": "pr: feature", "pull_request": map[string]any{"url": "…"}},
		{"number": 3, "title": "question"},
	}
}

func TestListIssues_FiltersPullRequestsByDefault(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/o/r/issues" {
			t.Errorf("path = %s, want /repos/o/r/issues", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(listIssuesMixedResponse())
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_issues"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_issues",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var items []map[string]any
	if err := json.Unmarshal(result.Data, &items); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items after filter, want 2 (PRs excluded)", len(items))
	}
	for _, item := range items {
		if _, isPR := item["pull_request"]; isPR {
			t.Errorf("PR leaked through filter: %v", item)
		}
	}
}

func TestListIssues_IncludePullRequestsPassesThrough(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(listIssuesMixedResponse())
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_issues"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_issues",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","include_pull_requests":true}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var items []map[string]any
	if err := json.Unmarshal(result.Data, &items); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3 (PRs included)", len(items))
	}
}

func TestListIssues_PassesFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != "closed" {
			t.Errorf("state = %q, want closed", q.Get("state"))
		}
		if q.Get("labels") != "bug,urgent" {
			t.Errorf("labels = %q, want bug,urgent", q.Get("labels"))
		}
		if q.Get("sort") != "updated" {
			t.Errorf("sort = %q, want updated", q.Get("sort"))
		}
		if q.Get("direction") != "desc" {
			t.Errorf("direction = %q, want desc", q.Get("direction"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_issues"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "github.list_issues",
		Parameters: json.RawMessage(
			`{"owner":"o","repo":"r","state":"closed","labels":"bug,urgent","sort":"updated","direction":"desc"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListIssues_InvalidFilters(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.list_issues"]

	tests := []struct {
		name   string
		params string
	}{
		{"invalid state", `{"owner":"o","repo":"r","state":"bogus"}`},
		{"invalid sort", `{"owner":"o","repo":"r","sort":"bogus"}`},
		{"invalid direction", `{"owner":"o","repo":"r","direction":"bogus"}`},
		{"missing owner", `{"repo":"r"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.list_issues",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
