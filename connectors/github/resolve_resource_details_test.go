package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testGitHubResolveServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *GitHubConnector) {
	t.Helper()
	srv := httptest.NewServer(handler)
	conn := newForTest(srv.Client(), srv.URL)
	return srv, conn
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
}

func TestResolveResourceDetails_UnknownAction(t *testing.T) {
	t.Parallel()

	conn := New()
	params, _ := json.Marshal(map[string]string{"owner": "a", "repo": "b"})
	details, err := conn.ResolveResourceDetails(context.Background(), "github.create_issue", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details for unhandled action, got %v", details)
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
}
