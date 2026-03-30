package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListPullRequests_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/pulls" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("state = %q, want open", r.URL.Query().Get("state"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"number":   1,
				"title":    "Fix bug",
				"state":    "open",
				"html_url": "https://github.com/octocat/hello-world/pull/1",
				"draft":    false,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_pull_requests"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_pull_requests",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","state":"open"}`),
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

func TestListPullRequests_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.list_pull_requests"]

	tests := []struct {
		name   string
		params string
	}{
		{"invalid state", `{"owner":"o","repo":"r","state":"deleted"}`},
		{"invalid sort", `{"owner":"o","repo":"r","sort":"stars"}`},
		{"invalid direction", `{"owner":"o","repo":"r","direction":"sideways"}`},
		{"per_page too large", `{"owner":"o","repo":"r","per_page":101}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.list_pull_requests",
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

func TestListPullRequests_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.list_pull_requests"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"hello-world"}`},
		{"missing repo", `{"owner":"octocat"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.list_pull_requests",
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
