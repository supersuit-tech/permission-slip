package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddReviewer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/pulls/42/requested_reviewers" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/pulls/42/requested_reviewers", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		reviewers, ok := reqBody["reviewers"].([]any)
		if !ok || len(reviewers) != 2 {
			t.Errorf("reviewers = %v, want [\"alice\",\"bob\"]", reqBody["reviewers"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"number":   42,
			"url":      "https://api.github.com/repos/octocat/hello-world/pulls/42",
			"html_url": "https://github.com/octocat/hello-world/pull/42",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.add_reviewer"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.add_reviewer",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":42,"reviewers":["alice","bob"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["number"] != float64(42) {
		t.Errorf("number = %v, want 42", data["number"])
	}
}

func TestAddReviewer_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.add_reviewer"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","pull_number":1,"reviewers":["alice"]}`},
		{"missing repo", `{"owner":"o","pull_number":1,"reviewers":["alice"]}`},
		{"missing pull_number", `{"owner":"o","repo":"r","reviewers":["alice"]}`},
		{"missing reviewers", `{"owner":"o","repo":"r","pull_number":1}`},
		{"empty reviewers", `{"owner":"o","repo":"r","pull_number":1,"reviewers":[]}`},
		{"empty string in reviewers", `{"owner":"o","repo":"r","pull_number":1,"reviewers":[""]}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.add_reviewer",
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
