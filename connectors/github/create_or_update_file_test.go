package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateOrUpdateFile_Create(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/contents/newfile.txt" {
			t.Errorf("path = %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if reqBody["message"] != "add file" {
			t.Errorf("message = %q", reqBody["message"])
		}
		if reqBody["content"] != "aGVsbG8=" {
			t.Errorf("content = %q", reqBody["content"])
		}
		if _, ok := reqBody["sha"]; ok {
			t.Error("sha should not be present for new file")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"content": map[string]any{
				"name":     "newfile.txt",
				"path":     "newfile.txt",
				"sha":      "def456",
				"html_url": "https://github.com/octocat/hello-world/blob/main/newfile.txt",
			},
			"commit": map[string]any{
				"sha":     "commit123",
				"message": "add file",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_or_update_file"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_or_update_file",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":"newfile.txt","message":"add file","content":"aGVsbG8="}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	content, ok := data["content"].(map[string]any)
	if !ok {
		t.Fatalf("content not a map")
	}
	if content["name"] != "newfile.txt" {
		t.Errorf("content.name = %v", content["name"])
	}
}

func TestCreateOrUpdateFile_Update_IncludesSHA(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if reqBody["sha"] != "existing_sha" {
			t.Errorf("sha = %q, want %q", reqBody["sha"], "existing_sha")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"content": map[string]any{"name": "README.md", "path": "README.md", "sha": "new_sha"},
			"commit":  map[string]any{"sha": "commit456", "message": "update"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_or_update_file"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_or_update_file",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","path":"README.md","message":"update","content":"dXBkYXRlZA==","sha":"existing_sha"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateOrUpdateFile_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_or_update_file"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","path":"f","message":"m","content":"c"}`},
		{"missing repo", `{"owner":"o","path":"f","message":"m","content":"c"}`},
		{"missing path", `{"owner":"o","repo":"r","message":"m","content":"c"}`},
		{"missing message", `{"owner":"o","repo":"r","path":"f","content":"c"}`},
		{"missing content", `{"owner":"o","repo":"r","path":"f","message":"m"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_or_update_file",
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

func TestCreateOrUpdateFile_PathInjection(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_or_update_file"]

	tests := []struct {
		name string
		path string
	}{
		{"query string injection", "file.txt?ref=evil"},
		{"fragment injection", "file.txt#evil"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := json.RawMessage(`{"owner":"o","repo":"r","path":"` + tt.path + `","message":"m","content":"Yg=="}`)
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_or_update_file",
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
