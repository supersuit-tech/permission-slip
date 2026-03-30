package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/issues" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/issues", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer ghp_test123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer ghp_test123")
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept = %q, want %q", got, "application/vnd.github+json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["title"] != "Found a bug" {
			t.Errorf("title = %q, want %q", reqBody["title"], "Found a bug")
		}
		if reqBody["body"] != "Something is broken" {
			t.Errorf("body = %q, want %q", reqBody["body"], "Something is broken")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"number":   42,
			"url":      "https://api.github.com/repos/octocat/hello-world/issues/42",
			"html_url": "https://github.com/octocat/hello-world/issues/42",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Found a bug","body":"Something is broken"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["number"] != float64(42) {
		t.Errorf("number = %v, want 42", data["number"])
	}
	if data["html_url"] != "https://github.com/octocat/hello-world/issues/42" {
		t.Errorf("html_url = %v, want https://github.com/octocat/hello-world/issues/42", data["html_url"])
	}
}

func TestCreateIssue_BodyOptional(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if _, ok := reqBody["body"]; ok {
			t.Error("body field should not be present when empty")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"number":   1,
			"url":      "https://api.github.com/repos/octocat/hello-world/issues/1",
			"html_url": "https://github.com/octocat/hello-world/issues/1",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"No body"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing owner",
			params: `{"repo":"hello-world","title":"Bug"}`,
		},
		{
			name:   "missing repo",
			params: `{"owner":"octocat","title":"Bug"}`,
		},
		{
			name:   "missing title",
			params: `{"owner":"octocat","repo":"hello-world"}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_issue",
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

func TestCreateIssue_GitHubAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Internal Server Error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCreateIssue_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	// Use a context deadline so both the client and the httptest server
	// see the cancellation, allowing a clean shutdown.
	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestCreateIssue_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "API rate limit exceeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 60*time.Second {
			t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
		}
	}
}

func TestCreateIssue_RateLimit403_RetryAfter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "You have exceeded a secondary rate limit",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 120*time.Second {
			t.Errorf("RetryAfter = %v, want 120s", rle.RetryAfter)
		}
	}
}

func TestCreateIssue_RateLimit403_XRateLimitRemaining(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "API rate limit exceeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCreateIssue_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad credentials",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Bug"}`),
		Credentials: connectors.NewCredentials(map[string]string{"api_key": "bad_token"}),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
