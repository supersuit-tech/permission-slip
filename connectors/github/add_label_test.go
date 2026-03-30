package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddLabel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/issues/5/labels" {
			t.Errorf("path = %s, want /repos/octocat/hello-world/issues/5/labels", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		labels, ok := reqBody["labels"].([]any)
		if !ok || len(labels) != 2 {
			t.Errorf("labels = %v, want [\"bug\",\"urgent\"]", reqBody["labels"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "name": "bug"},
			{"id": 2, "name": "urgent"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.add_label"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.add_label",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","issue_number":5,"labels":["bug","urgent"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(data) != 2 {
		t.Errorf("got %d labels, want 2", len(data))
	}
}

func TestAddLabel_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.add_label"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","issue_number":1,"labels":["bug"]}`},
		{"missing repo", `{"owner":"o","issue_number":1,"labels":["bug"]}`},
		{"missing issue_number", `{"owner":"o","repo":"r","labels":["bug"]}`},
		{"missing labels", `{"owner":"o","repo":"r","issue_number":1}`},
		{"empty labels", `{"owner":"o","repo":"r","issue_number":1,"labels":[]}`},
		{"empty string in labels", `{"owner":"o","repo":"r","issue_number":1,"labels":[""]}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.add_label",
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
