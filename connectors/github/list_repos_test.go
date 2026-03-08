package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListRepos_UserRepos(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repos" {
			t.Errorf("path = %s, want /user/repos", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "repo1", "full_name": "octocat/repo1", "private": false, "html_url": "https://github.com/octocat/repo1"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_repos"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_repos",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(data) != 1 {
		t.Errorf("len(data) = %d, want 1", len(data))
	}
}

func TestListRepos_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.list_repos"]

	tests := []struct {
		name   string
		params string
	}{
		{"invalid type", `{"type":"invalid"}`},
		{"invalid visibility", `{"visibility":"unknown"}`},
		{"invalid sort", `{"sort":"badfield"}`},
		{"per_page too large", `{"per_page":101}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.list_repos",
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

func TestListRepos_OrgRepos(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/my-org/repos" {
			t.Errorf("path = %s, want /orgs/my-org/repos", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_repos"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_repos",
		Parameters:  json.RawMessage(`{"org":"my-org"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
