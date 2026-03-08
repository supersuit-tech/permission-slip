package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateWebhook_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/hooks" {
			t.Errorf("path = %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if reqBody["name"] != "web" {
			t.Errorf("name = %v, want web", reqBody["name"])
		}
		config, ok := reqBody["config"].(map[string]any)
		if !ok {
			t.Fatal("config not present or not a map")
		}
		if config["url"] != "https://example.com/webhook" {
			t.Errorf("config.url = %v", config["url"])
		}
		if config["content_type"] != "json" {
			t.Errorf("config.content_type = %v, want json", config["content_type"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       1,
			"url":      "https://api.github.com/repos/octocat/hello-world/hooks/1",
			"html_url": "https://github.com/octocat/hello-world/settings/hooks/1",
			"active":   true,
			"events":   []string{"push", "pull_request"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_webhook"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_webhook",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","url":"https://example.com/webhook","events":["push","pull_request"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != float64(1) {
		t.Errorf("id = %v, want 1", data["id"])
	}
}

func TestCreateWebhook_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_webhook"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","url":"https://example.com","events":["push"]}`},
		{"missing repo", `{"owner":"o","url":"https://example.com","events":["push"]}`},
		{"missing url", `{"owner":"o","repo":"r","events":["push"]}`},
		{"missing events", `{"owner":"o","repo":"r","url":"https://example.com"}`},
		{"empty events", `{"owner":"o","repo":"r","url":"https://example.com","events":[]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_webhook",
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
