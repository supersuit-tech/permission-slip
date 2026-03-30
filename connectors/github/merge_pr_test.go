package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestMergePR_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/pulls/42/merge" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/pulls/42/merge", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer ghp_test123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer ghp_test123")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["merge_method"] != "squash" {
			t.Errorf("merge_method = %q, want %q", reqBody["merge_method"], "squash")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"sha":     "abc123def456",
			"merged":  true,
			"message": "Pull Request successfully merged",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.merge_pr"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.merge_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":42,"merge_method":"squash"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["sha"] != "abc123def456" {
		t.Errorf("sha = %v, want abc123def456", data["sha"])
	}
	if data["merged"] != true {
		t.Errorf("merged = %v, want true", data["merged"])
	}
}

func TestMergePR_DefaultMergeMethod(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["merge_method"] != "merge" {
			t.Errorf("merge_method = %q, want %q (default)", reqBody["merge_method"], "merge")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"sha":     "abc123",
			"merged":  true,
			"message": "Pull Request successfully merged",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.merge_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.merge_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":1}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestMergePR_PRNotMergeable(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Pull Request is not mergeable",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.merge_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.merge_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":42}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestMergePR_GitHubAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Head branch was modified. Review and try the merge again.",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.merge_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.merge_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":42}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestMergePR_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.merge_pr"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing owner",
			params: `{"repo":"hello-world","pull_number":1}`,
		},
		{
			name:   "missing repo",
			params: `{"owner":"octocat","pull_number":1}`,
		},
		{
			name:   "missing pull_number",
			params: `{"owner":"octocat","repo":"hello-world"}`,
		},
		{
			name:   "zero pull_number",
			params: `{"owner":"octocat","repo":"hello-world","pull_number":0}`,
		},
		{
			name:   "invalid merge_method",
			params: `{"owner":"octocat","repo":"hello-world","pull_number":1,"merge_method":"invalid"}`,
		},
		{
			name:   "invalid JSON",
			params: `{bad}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.merge_pr",
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

func TestMergePR_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Resource not accessible by integration",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.merge_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.merge_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":42}`),
		Credentials: connectors.NewCredentials(map[string]string{"api_key": "bad_token"}),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
