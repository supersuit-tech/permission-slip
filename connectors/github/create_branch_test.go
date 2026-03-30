package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateBranch_Success(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)

		if n == 1 {
			// First call: resolve ref
			if r.Method != http.MethodGet {
				t.Errorf("ref resolve method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/repos/octocat/hello-world/git/ref/heads/main" {
				t.Errorf("ref path = %s, want /repos/octocat/hello-world/git/ref/heads/main", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"ref": "refs/heads/main",
				"object": map[string]string{
					"sha": "abc123def456",
				},
			})
			return
		}

		// Second call: create ref
		if r.Method != http.MethodPost {
			t.Errorf("create ref method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/git/refs" {
			t.Errorf("create ref path = %s, want /repos/octocat/hello-world/git/refs", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["ref"] != "refs/heads/feature-branch" {
			t.Errorf("ref = %q, want %q", reqBody["ref"], "refs/heads/feature-branch")
		}
		if reqBody["sha"] != "abc123def456" {
			t.Errorf("sha = %q, want %q", reqBody["sha"], "abc123def456")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"ref": "refs/heads/feature-branch",
			"url": "https://api.github.com/repos/octocat/hello-world/git/refs/heads/feature-branch",
			"object": map[string]string{
				"sha": "abc123def456",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_branch"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_branch",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","branch_name":"feature-branch","from_ref":"main"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["ref"] != "refs/heads/feature-branch" {
		t.Errorf("ref = %v, want refs/heads/feature-branch", data["ref"])
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls (resolve ref + create ref), got %d", callCount.Load())
	}
}

func TestCreateBranch_FromTag(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)

		if n == 1 {
			// Verify full ref path is used as-is (not prefixed with heads/)
			if r.URL.Path != "/repos/octocat/hello-world/git/ref/tags/v1.0" {
				t.Errorf("ref path = %s, want /repos/octocat/hello-world/git/ref/tags/v1.0", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"ref":    "refs/tags/v1.0",
				"object": map[string]string{"sha": "abc123"},
			})
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"ref":    "refs/heads/hotfix",
			"object": map[string]string{"sha": "abc123"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_branch"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_branch",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","branch_name":"hotfix","from_ref":"tags/v1.0"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateBranch_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_branch"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","branch_name":"b","from_ref":"main"}`},
		{"missing repo", `{"owner":"o","branch_name":"b","from_ref":"main"}`},
		{"missing branch_name", `{"owner":"o","repo":"r","from_ref":"main"}`},
		{"missing from_ref", `{"owner":"o","repo":"r","branch_name":"b"}`},
		{"branch_name with path traversal", `{"owner":"o","repo":"r","branch_name":"../../main","from_ref":"main"}`},
		{"branch_name starts with dot", `{"owner":"o","repo":"r","branch_name":".hidden","from_ref":"main"}`},
		{"branch_name with tilde", `{"owner":"o","repo":"r","branch_name":"foo~1","from_ref":"main"}`},
		{"from_ref with path traversal", `{"owner":"o","repo":"r","branch_name":"b","from_ref":"heads/../main"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_branch",
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
