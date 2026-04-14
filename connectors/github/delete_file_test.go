package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/contents/docs/OLD.md" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/contents/docs/OLD.md", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		if reqBody["message"] != "cleanup" {
			t.Errorf("message = %q, want cleanup", reqBody["message"])
		}
		if reqBody["sha"] != "abc123" {
			t.Errorf("sha = %q, want abc123", reqBody["sha"])
		}
		if reqBody["branch"] != "main" {
			t.Errorf("branch = %q, want main", reqBody["branch"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"commit": map[string]any{"sha": "def456"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.delete_file"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "github.delete_file",
		Parameters: json.RawMessage(
			`{"owner":"octocat","repo":"hello-world","path":"docs/OLD.md","message":"cleanup","sha":"abc123","branch":"main"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
}

func TestDeleteFile_OmitsBranchWhenEmpty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		if _, ok := reqBody["branch"]; ok {
			t.Errorf("branch should not be present when omitted, got %q", reqBody["branch"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"commit": map[string]any{"sha": "d"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.delete_file"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "github.delete_file",
		Parameters: json.RawMessage(
			`{"owner":"o","repo":"r","path":"a.txt","message":"m","sha":"s"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestDeleteFile_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.delete_file"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","path":"a","message":"m","sha":"s"}`},
		{"missing path", `{"owner":"o","repo":"r","message":"m","sha":"s"}`},
		{"missing message", `{"owner":"o","repo":"r","path":"a","sha":"s"}`},
		{"missing sha", `{"owner":"o","repo":"r","path":"a","message":"m"}`},
		{"absolute path", `{"owner":"o","repo":"r","path":"/a","message":"m","sha":"s"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.delete_file",
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
