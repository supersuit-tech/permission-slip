package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestClosePR_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/pulls/42" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/pulls/42", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		if reqBody["state"] != "closed" {
			t.Errorf("state = %q, want closed", reqBody["state"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 42, "state": "closed",
			"html_url": "https://github.com/octocat/hello-world/pull/42",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.close_pr"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.close_pr",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","pull_number":42}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["state"] != "closed" {
		t.Errorf("state = %v, want closed", data["state"])
	}
}

func TestClosePR_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.close_pr"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","pull_number":1}`},
		{"missing repo", `{"owner":"o","pull_number":1}`},
		{"missing pull_number", `{"owner":"o","repo":"r"}`},
		{"zero pull_number", `{"owner":"o","repo":"r","pull_number":0}`},
		{"invalid JSON", `{invalid}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.close_pr",
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
