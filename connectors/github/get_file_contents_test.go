package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetFileContents_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/contents/README.md" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"name":     "README.md",
			"path":     "README.md",
			"sha":      "abc123",
			"size":     100,
			"content":  "SGVsbG8gV29ybGQ=",
			"html_url": "https://github.com/octocat/hello-world/blob/main/README.md",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.get_file_contents"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.get_file_contents",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":"README.md"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["name"] != "README.md" {
		t.Errorf("name = %v, want README.md", data["name"])
	}
	if data["content"] != "SGVsbG8gV29ybGQ=" {
		t.Errorf("content = %v", data["content"])
	}
}

func TestGetFileContents_WithRef(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "develop" {
			t.Errorf("ref = %q, want %q", r.URL.Query().Get("ref"), "develop")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"name": "README.md",
			"path": "README.md",
			"sha":  "abc123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.get_file_contents"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.get_file_contents",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":"README.md","ref":"develop"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetFileContents_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.get_file_contents"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"hello-world","path":"README.md"}`},
		{"missing repo", `{"owner":"octocat","path":"README.md"}`},
		{"missing path", `{"owner":"octocat","repo":"hello-world"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.get_file_contents",
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
