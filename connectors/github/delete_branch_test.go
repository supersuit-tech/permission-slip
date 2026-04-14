package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteBranch_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/git/refs/heads/feature/new-ui" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/git/refs/heads/feature/new-ui", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.delete_branch"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.delete_branch",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","branch_name":"feature/new-ui"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestDeleteBranch_EscapesSpecialCharsPerSegment(t *testing.T) {
	t.Parallel()

	// A branch name with characters that need encoding ("%" and space) must
	// still be URL-safe — but the "/" separators must be preserved so the API
	// treats it as a multi-segment ref.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is already decoded by net/http, so we verify via the
		// raw path to ensure each segment was encoded independently.
		want := "/repos/o/r/git/refs/heads/release%2Fv1/spaces%20and%25"
		if r.URL.EscapedPath() != want {
			t.Errorf("escaped path = %s, want %s", r.URL.EscapedPath(), want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.delete_branch"]

	// Note: "release/v1/spaces and%" is a single branch name from the user's
	// perspective — the `/` inside is part of the name, not a path separator.
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.delete_branch",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","branch_name":"release/v1/spaces and%"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestDeleteBranch_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.delete_branch"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","branch_name":"main"}`},
		{"missing repo", `{"owner":"o","branch_name":"main"}`},
		{"missing branch_name", `{"owner":"o","repo":"r"}`},
		{"empty branch_name", `{"owner":"o","repo":"r","branch_name":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.delete_branch",
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
