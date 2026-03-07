package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateRelease_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/releases" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/releases", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["tag_name"] != "v1.0.0" {
			t.Errorf("tag_name = %q, want %q", reqBody["tag_name"], "v1.0.0")
		}
		if reqBody["name"] != "Release 1.0" {
			t.Errorf("name = %q, want %q", reqBody["name"], "Release 1.0")
		}
		if reqBody["prerelease"] != true {
			t.Errorf("prerelease = %v, want true", reqBody["prerelease"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       1,
			"url":      "https://api.github.com/repos/octocat/hello-world/releases/1",
			"html_url": "https://github.com/octocat/hello-world/releases/tag/v1.0.0",
			"tag_name": "v1.0.0",
			"name":     "Release 1.0",
			"draft":    false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_release"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_release",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","tag_name":"v1.0.0","name":"Release 1.0","body":"Notes","prerelease":true}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["tag_name"] != "v1.0.0" {
		t.Errorf("tag_name = %v, want v1.0.0", data["tag_name"])
	}
}

func TestCreateRelease_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_release"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","tag_name":"v1.0.0"}`},
		{"missing repo", `{"owner":"o","tag_name":"v1.0.0"}`},
		{"missing tag_name", `{"owner":"o","repo":"r"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_release",
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

func TestCreateRelease_OptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if _, ok := reqBody["name"]; ok {
			t.Error("name field should not be present when empty")
		}
		if _, ok := reqBody["body"]; ok {
			t.Error("body field should not be present when empty")
		}
		if _, ok := reqBody["draft"]; ok {
			t.Error("draft field should not be present when false")
		}
		if _, ok := reqBody["prerelease"]; ok {
			t.Error("prerelease field should not be present when false")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       1,
			"url":      "https://api.github.com/repos/o/r/releases/1",
			"html_url": "https://github.com/o/r/releases/tag/v1.0.0",
			"tag_name": "v1.0.0",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_release"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_release",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","tag_name":"v1.0.0"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
