package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddComment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/issues/7/comments" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/issues/7/comments", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["body"] != "Looks good!" {
			t.Errorf("body = %q, want %q", reqBody["body"], "Looks good!")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       123,
			"url":      "https://api.github.com/repos/octocat/hello-world/issues/comments/123",
			"html_url": "https://github.com/octocat/hello-world/issues/7#issuecomment-123",
			"body":     "Looks good!",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.add_comment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.add_comment",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","issue_number":7,"body":"Looks good!"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != float64(123) {
		t.Errorf("id = %v, want 123", data["id"])
	}
	if data["body"] != "Looks good!" {
		t.Errorf("body = %v, want Looks good!", data["body"])
	}
}

func TestAddComment_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.add_comment"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","issue_number":1,"body":"hi"}`},
		{"missing repo", `{"owner":"o","issue_number":1,"body":"hi"}`},
		{"missing issue_number", `{"owner":"o","repo":"r","body":"hi"}`},
		{"missing body", `{"owner":"o","repo":"r","issue_number":1}`},
		{"empty body", `{"owner":"o","repo":"r","issue_number":1,"body":""}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.add_comment",
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
