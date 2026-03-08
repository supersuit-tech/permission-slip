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
	// "SGVsbG8gV29ybGQ=" is base64 for "Hello World"
	if data["decoded_content"] != "Hello World" {
		t.Errorf("decoded_content = %v, want \"Hello World\"", data["decoded_content"])
	}
}

func TestGetFileContents_DecodedContentWithNewlines(t *testing.T) {
	t.Parallel()

	// GitHub wraps base64 at 60 chars — simulate that here.
	// "Hello World" = "SGVsbG8gV29ybGQ=" but GitHub would wrap longer content.
	// "Hello\nWorld\n" in base64 is "SGVsbG8KV29ybGQK"
	wrappedBase64 := "SGVsbG8K\nV29ybGQK\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "file.txt",
			"path":    "file.txt",
			"sha":     "abc",
			"content": wrappedBase64,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.get_file_contents"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.get_file_contents",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":"file.txt"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["decoded_content"] != "Hello\nWorld\n" {
		t.Errorf("decoded_content = %q, want %q", data["decoded_content"], "Hello\nWorld\n")
	}
}

func TestGetFileContents_BinaryFileEmptyDecodedContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Invalid UTF-8 bytes as base64 → decoded_content should be empty.
		json.NewEncoder(w).Encode(map[string]any{
			"name":    "image.png",
			"path":    "image.png",
			"sha":     "abc",
			"content": "/9j/4AAQ", // truncated JPEG header bytes — not valid UTF-8
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.get_file_contents"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.get_file_contents",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":"image.png"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["decoded_content"] != "" {
		t.Errorf("decoded_content = %q for binary file, want empty string", data["decoded_content"])
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

func TestGetFileContents_PathInjection(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.get_file_contents"]

	tests := []struct {
		name string
		path string
	}{
		{"query string injection", "README.md?ref=evil"},
		{"fragment injection", "README.md#evil"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":` + `"` + tt.path + `"` + `}`)
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.get_file_contents",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error for path injection, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
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
			t.Parallel()
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
