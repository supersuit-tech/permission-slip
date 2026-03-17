package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateRepo_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/user/repos" {
			t.Errorf("path = %s, want /user/repos", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer ghp_test123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer ghp_test123")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["name"] != "my-new-repo" {
			t.Errorf("name = %q, want %q", reqBody["name"], "my-new-repo")
		}
		if reqBody["private"] != true {
			t.Errorf("private = %v, want true", reqBody["private"])
		}
		if reqBody["description"] != "A test repo" {
			t.Errorf("description = %q, want %q", reqBody["description"], "A test repo")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":          12345,
			"name":        "my-new-repo",
			"full_name":   "octocat/my-new-repo",
			"private":     true,
			"html_url":    "https://github.com/octocat/my-new-repo",
			"clone_url":   "https://github.com/octocat/my-new-repo.git",
			"description": "A test repo",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"my-new-repo","private":true,"description":"A test repo"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["full_name"] != "octocat/my-new-repo" {
		t.Errorf("full_name = %v, want octocat/my-new-repo", data["full_name"])
	}
	if data["html_url"] != "https://github.com/octocat/my-new-repo" {
		t.Errorf("html_url = %v, want https://github.com/octocat/my-new-repo", data["html_url"])
	}
}

func TestCreateRepo_OrgSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/my-org/repos" {
			t.Errorf("path = %s, want /orgs/my-org/repos", r.URL.Path)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":        67890,
			"name":      "org-repo",
			"full_name": "my-org/org-repo",
			"private":   false,
			"html_url":  "https://github.com/my-org/org-repo",
			"clone_url": "https://github.com/my-org/org-repo.git",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"org-repo","org":"my-org"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["full_name"] != "my-org/org-repo" {
		t.Errorf("full_name = %v, want my-org/org-repo", data["full_name"])
	}
}

func TestCreateRepo_NameTrimmed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		// Verify the trimmed name reaches the API, not the padded original.
		if reqBody["name"] != "my-repo" {
			t.Errorf("name = %q, want %q (should be trimmed)", reqBody["name"], "my-repo")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":        1,
			"name":      "my-repo",
			"full_name": "octocat/my-repo",
			"html_url":  "https://github.com/octocat/my-repo",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"  my-repo  "}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateRepo_DescriptionOptional(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if _, ok := reqBody["description"]; ok {
			t.Error("description field should not be present when empty")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":        1,
			"name":      "bare-repo",
			"full_name": "octocat/bare-repo",
			"html_url":  "https://github.com/octocat/bare-repo",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"bare-repo"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateRepo_WhitespaceDescription(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if _, ok := reqBody["description"]; ok {
			t.Error("description field should not be present when whitespace-only")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":        1,
			"name":      "my-repo",
			"full_name": "octocat/my-repo",
			"html_url":  "https://github.com/octocat/my-repo",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"my-repo","description":"   "}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateRepo_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_repo"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing name",
			params: `{"org":"my-org"}`,
		},
		{
			name:   "empty name",
			params: `{"name":""}`,
		},
		{
			name:   "whitespace-only name",
			params: `{"name":"   "}`,
		},
		{
			name:   "dot-only name",
			params: `{"name":"."}`,
		},
		{
			name:   "double-dot name",
			params: `{"name":".."}`,
		},
		{
			name:   "name ending in .git",
			params: `{"name":"myrepo.git"}`,
		},
		{
			name:   "invalid characters in name",
			params: `{"name":"my repo/here"}`,
		},
		{
			name:   "name exceeds 100 characters",
			params: `{"name":"aaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeaaaaaaaaaabbbbbbbbbbccccccccccddddddddddeeeeeeeeeeX"}`,
		},
		{
			name:   "invalid org name with spaces and slashes",
			params: `{"name":"valid-repo","org":"bad org/name"}`,
		},
		{
			name:   "invalid org name with dots",
			params: `{"name":"valid-repo","org":"my.company"}`,
		},
		{
			name:   "invalid org name with underscores",
			params: `{"name":"valid-repo","org":"my_org"}`,
		},
		{
			name:   "invalid org name leading hyphen",
			params: `{"name":"valid-repo","org":"-leading"}`,
		},
		{
			name:   "invalid org name trailing hyphen",
			params: `{"name":"valid-repo","org":"trailing-"}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_repo",
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

func TestCreateRepo_OrgNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Not Found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"test-repo","org":"nonexistent-org"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
	// Should mention the org name in the error for debugging.
	if got := err.Error(); !strings.Contains(got, "nonexistent-org") {
		t.Errorf("error should mention org name, got: %s", got)
	}
}

func TestCreateRepo_PersonalNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Not Found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"test-repo"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	// Personal repo 404 should still be a ValidationError (from checkResponse).
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateRepo_GitHubAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"test-repo"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCreateRepo_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad credentials",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"test-repo"}`),
		Credentials: connectors.NewCredentials(map[string]string{"api_key": "bad_token"}),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCreateRepo_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_repo"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "github.create_repo",
		Parameters:  json.RawMessage(`{"name":"test-repo"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
