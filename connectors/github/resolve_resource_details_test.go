package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func testGitHubResolveServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GitHubConnector) {
	t.Helper()
	srv := httptest.NewServer(handler)
	conn := newForTest(srv.Client(), srv.URL)
	return srv, conn
}

func assertOwnerRepoOverlay(t *testing.T, details map[string]any, owner, repo string) {
	t.Helper()
	if details == nil {
		t.Fatal("expected resource details")
	}
	if details["owner_name"] != owner {
		t.Errorf("owner_name: want %q, got %v", owner, details["owner_name"])
	}
	wantRepo := owner + "/" + repo
	if details["repo_name"] != wantRepo {
		t.Errorf("repo_name: want %q, got %v", wantRepo, details["repo_name"])
	}
	wantOwnerURL := "https://github.com/" + owner
	if details["owner_url"] != wantOwnerURL {
		t.Errorf("owner_url: want %q, got %v", wantOwnerURL, details["owner_url"])
	}
	wantRepoURL := "https://github.com/" + owner + "/" + repo
	if details["repo_url"] != wantRepoURL {
		t.Errorf("repo_url: want %q, got %v", wantRepoURL, details["repo_url"])
	}
}

func TestResolveResourceDetails_Workflow(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		want := "/repos/acme/app/actions/workflows/deploy.yml"
		if r.URL.Path != want {
			t.Errorf("expected path %q, got %q", want, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "Deploy to production"})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{
		"owner":       "acme",
		"repo":        "app",
		"workflow_id": "deploy.yml",
		"ref":         "main",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.trigger_workflow", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["workflow_name"] != "Deploy to production" {
		t.Errorf("expected workflow_name, got %v", details["workflow_name"])
	}
	assertOwnerRepoOverlay(t, details, "acme", "app")
}

func TestResolveResourceDetails_Webhook(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		want := "/repos/acme/app/hooks/42"
		if r.URL.Path != want {
			t.Errorf("expected path %q, got %q", want, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]string{"url": "https://example.com/webhook"},
			"events": []string{"push", "pull_request"},
		})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]any{
		"owner":   "acme",
		"repo":    "app",
		"hook_id": 42,
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.delete_webhook", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["webhook_url"] != "https://example.com/webhook" {
		t.Errorf("expected webhook_url, got %v", details["webhook_url"])
	}
	if details["webhook_events"] != "push, pull_request" {
		t.Errorf("expected webhook_events, got %v", details["webhook_events"])
	}
	assertOwnerRepoOverlay(t, details, "acme", "app")
}

func TestResolveResourceDetails_UnknownAction(t *testing.T) {
	t.Parallel()

	conn := New()
	params, _ := json.Marshal(map[string]any{
		"owner":        "a",
		"repo":         "b",
		"issue_number": 12,
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.create_issue", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertOwnerRepoOverlay(t, details, "a", "b")
	if details["issue_number_name"] != "a/b#12" {
		t.Errorf("issue_number_name: want a/b#12, got %v", details["issue_number_name"])
	}
	if details["issue_number_url"] != "https://github.com/a/b/issues/12" {
		t.Errorf("issue_number_url: want issues URL, got %v", details["issue_number_url"])
	}
}

func TestResolveResourceDetails_UnknownAction_NoOwnerRepo(t *testing.T) {
	t.Parallel()

	conn := New()
	details, err := conn.ResolveResourceDetails(context.Background(), "github.create_issue", []byte(`{}`), validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details with no owner/repo, got %v", details)
	}
}

func TestResolveResourceDetails_WorkflowMissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	params, _ := json.Marshal(map[string]string{"owner": "a", "repo": "b", "ref": "main"})
	_, err := conn.ResolveResourceDetails(context.Background(), "github.trigger_workflow", params, validCreds())
	if err == nil {
		t.Fatal("expected error for missing workflow_id")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestResolveResourceDetails_Workflow_EmptyName(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"name": "   "})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{
		"owner": "acme", "repo": "app", "workflow_id": "deploy.yml", "ref": "main",
	})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.trigger_workflow", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := details["workflow_name"]; ok {
		t.Errorf("expected no workflow_name for empty API name, got %v", details["workflow_name"])
	}
	assertOwnerRepoOverlay(t, details, "acme", "app")
}

func TestResolveResourceDetails_Webhook_EmptyURLAndEvents(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"config": map[string]string{}, "events": []string{}})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "hook_id": 42})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.delete_webhook", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := details["webhook_url"]; ok {
		t.Errorf("expected no webhook_url when API returns none, got %v", details["webhook_url"])
	}
	assertOwnerRepoOverlay(t, details, "acme", "app")
}

func TestResolveResourceDetails_Webhook_ManyEventsTruncated(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]string{"url": "https://example.com/h"},
			"events": []string{"push", "pull_request", "issues", "workflow_dispatch", "release"},
		})
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "hook_id": 1})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.delete_webhook", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "push, pull_request, issues, +2 more"
	if details["webhook_events"] != want {
		t.Errorf("webhook_events: want %q, got %v", want, details["webhook_events"])
	}
}

func TestResolveResourceDetails_Workflow_APIError(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{
		"owner": "acme", "repo": "app", "workflow_id": "missing.yml", "ref": "main",
	})
	_, err := conn.ResolveResourceDetails(context.Background(), "github.trigger_workflow", params, validCreds())
	if err == nil {
		t.Fatal("expected error for 404 API response")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestResolveResourceDetails_Webhook_APIError(t *testing.T) {
	t.Parallel()

	srv, conn := testGitHubResolveServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"Server Error"}`))
	}))
	defer srv.Close()

	params, _ := json.Marshal(map[string]any{"owner": "acme", "repo": "app", "hook_id": 99})
	_, err := conn.ResolveResourceDetails(context.Background(), "github.delete_webhook", params, validCreds())
	if err == nil {
		t.Fatal("expected error for 500 API response")
	}
	if !connectors.IsExternalError(err) {
		t.Fatalf("expected ExternalError for 500, got %T: %v", err, err)
	}
}
