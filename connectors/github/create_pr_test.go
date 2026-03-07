package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePR_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/pulls" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/pulls", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["title"] != "Add feature" {
			t.Errorf("title = %q, want %q", reqBody["title"], "Add feature")
		}
		if reqBody["head"] != "feature-branch" {
			t.Errorf("head = %q, want %q", reqBody["head"], "feature-branch")
		}
		if reqBody["base"] != "main" {
			t.Errorf("base = %q, want %q", reqBody["base"], "main")
		}
		if reqBody["draft"] != true {
			t.Errorf("draft = %v, want true", reqBody["draft"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"number":   99,
			"url":      "https://api.github.com/repos/octocat/hello-world/pulls/99",
			"html_url": "https://github.com/octocat/hello-world/pull/99",
			"state":    "open",
			"draft":    true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_pr"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","title":"Add feature","head":"feature-branch","base":"main","draft":true}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["number"] != float64(99) {
		t.Errorf("number = %v, want 99", data["number"])
	}
	if data["html_url"] != "https://github.com/octocat/hello-world/pull/99" {
		t.Errorf("html_url = %v, want https://github.com/octocat/hello-world/pull/99", data["html_url"])
	}
}

func TestCreatePR_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.create_pr"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","title":"t","head":"h","base":"b"}`},
		{"missing repo", `{"owner":"o","title":"t","head":"h","base":"b"}`},
		{"missing title", `{"owner":"o","repo":"r","head":"h","base":"b"}`},
		{"missing head", `{"owner":"o","repo":"r","title":"t","base":"b"}`},
		{"missing base", `{"owner":"o","repo":"r","title":"t","head":"h"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.create_pr",
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

func TestCreatePR_BodyOptional(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if _, ok := reqBody["body"]; ok {
			t.Error("body field should not be present when empty")
		}
		if _, ok := reqBody["draft"]; ok {
			t.Error("draft field should not be present when false")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"number":   1,
			"url":      "https://api.github.com/repos/o/r/pulls/1",
			"html_url": "https://github.com/o/r/pull/1",
			"state":    "open",
			"draft":    false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.create_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.create_pr",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","title":"t","head":"h","base":"b"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
