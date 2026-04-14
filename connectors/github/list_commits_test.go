package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListCommits_SuccessWithQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/o/r/commits" {
			t.Errorf("path = %s, want /repos/o/r/commits", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("sha") != "main" {
			t.Errorf("sha = %q, want main", q.Get("sha"))
		}
		if q.Get("path") != "src/file.go" {
			t.Errorf("path = %q, want src/file.go", q.Get("path"))
		}
		if q.Get("author") != "octocat" {
			t.Errorf("author = %q, want octocat", q.Get("author"))
		}
		if q.Get("per_page") != "50" {
			t.Errorf("per_page = %q, want 50", q.Get("per_page"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{{"sha": "a"}, {"sha": "b"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_commits"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "github.list_commits",
		Parameters: json.RawMessage(
			`{"owner":"o","repo":"r","sha":"main","path":"src/file.go","author":"octocat","per_page":50}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	var commits []map[string]any
	if err := json.Unmarshal(result.Data, &commits); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("got %d commits, want 2", len(commits))
	}
}

func TestListCommits_OmitsEmptyQueryParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for _, k := range []string{"sha", "path", "author"} {
			if q.Has(k) {
				t.Errorf("%s should not be set when empty, got %q", k, q.Get(k))
			}
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_commits"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_commits",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListCommits_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.list_commits"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r"}`},
		{"missing repo", `{"owner":"o"}`},
		{"absolute path", `{"owner":"o","repo":"r","path":"/abs"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.list_commits",
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
