package asana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchTasks_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/workspaces/ws123/tasks/search" {
			t.Errorf("path = %s, want /workspaces/ws123/tasks/search", got)
		}

		q := r.URL.Query()
		if q.Get("text") != "login bug" {
			t.Errorf("text = %q, want %q", q.Get("text"), "login bug")
		}
		if q.Get("completed") != "false" {
			t.Errorf("completed = %q, want %q", q.Get("completed"), "false")
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q, want %q", q.Get("limit"), "10")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"gid": "1", "name": "Fix login bug", "completed": false, "permalink_url": "https://app.asana.com/0/1/1"},
				{"gid": "2", "name": "Login page error", "completed": false, "permalink_url": "https://app.asana.com/0/1/2"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.search_tasks"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.search_tasks",
		Parameters:  json.RawMessage(`{"workspace_id":"ws123","text":"login bug","completed":false,"limit":10}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("got %d results, want 2", len(data))
	}
	if data[0]["gid"] != "1" {
		t.Errorf("first result gid = %v, want %q", data[0]["gid"], "1")
	}
}

func TestSearchTasks_DefaultLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if q := r.URL.Query().Get("limit"); q != "20" {
			t.Errorf("limit = %q, want %q (default)", q, "20")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.search_tasks"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.search_tasks",
		Parameters:  json.RawMessage(`{"workspace_id":"ws123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchTasks_MissingWorkspaceID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.search_tasks"]

	// No workspace_id in params or credentials → should fail.
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.search_tasks",
		Parameters:  json.RawMessage(`{"text":"test"}`),
		Credentials: connectors.NewCredentials(map[string]string{"api_key": "tok"}),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchTasks_WorkspaceIDFromCredentials(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/workspaces/cred-ws-id/tasks/search" {
			t.Errorf("path = %s, want /workspaces/cred-ws-id/tasks/search", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.search_tasks"]

	// workspace_id not in params, but present in credentials.
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "asana.search_tasks",
		Parameters: json.RawMessage(`{}`),
		Credentials: connectors.NewCredentials(map[string]string{
			"api_key":      "0/abc123test",
			"workspace_id": "cred-ws-id",
		}),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchTasks_WithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("assignee.any") != "me" {
			t.Errorf("assignee.any = %q, want %q", q.Get("assignee.any"), "me")
		}
		if q.Get("due_on.before") != "2026-04-01" {
			t.Errorf("due_on.before = %q, want %q", q.Get("due_on.before"), "2026-04-01")
		}
		if q.Get("due_on.after") != "2026-03-01" {
			t.Errorf("due_on.after = %q, want %q", q.Get("due_on.after"), "2026-03-01")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.search_tasks"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.search_tasks",
		Parameters:  json.RawMessage(`{"workspace_id":"ws1","assignee":"me","due_on_before":"2026-04-01","due_on_after":"2026-03-01"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
