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

func TestCloseIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/issues/10" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/issues/10", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["state"] != "closed" {
			t.Errorf("state = %q, want %q", reqBody["state"], "closed")
		}
		if reqBody["state_reason"] != "completed" {
			t.Errorf("state_reason = %q, want %q", reqBody["state_reason"], "completed")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number":   10,
			"url":      "https://api.github.com/repos/octocat/hello-world/issues/10",
			"html_url": "https://github.com/octocat/hello-world/issues/10",
			"state":    "closed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.close_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.close_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","issue_number":10}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["state"] != "closed" {
		t.Errorf("state = %v, want closed", data["state"])
	}
}

func TestCloseIssue_WithComment(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)

		if n == 1 {
			// First call: comment
			if r.Method != http.MethodPost {
				t.Errorf("comment method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/repos/octocat/hello-world/issues/10/comments" {
				t.Errorf("comment path = %s, want /repos/octocat/hello-world/issues/10/comments", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]string
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Fatalf("unmarshaling comment body: %v", err)
			}
			if reqBody["body"] != "Closing this as done." {
				t.Errorf("comment body = %q, want %q", reqBody["body"], "Closing this as done.")
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
			return
		}

		// Second call: close
		if r.Method != http.MethodPatch {
			t.Errorf("close method = %s, want PATCH", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 10,
			"state":  "closed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.close_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.close_issue",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","issue_number":10,"comment":"Closing this as done."}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls (comment + close), got %d", callCount.Load())
	}
}

func TestCloseIssue_InvalidStateReason(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.close_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.close_issue",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","issue_number":1,"state_reason":"invalid"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCloseIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.close_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","issue_number":1}`},
		{"missing repo", `{"owner":"o","issue_number":1}`},
		{"missing issue_number", `{"owner":"o","repo":"r"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.close_issue",
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
