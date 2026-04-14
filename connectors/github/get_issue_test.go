package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/o/r/issues/7" {
			t.Errorf("path = %s, want /repos/o/r/issues/7", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 7, "title": "bug", "state": "open",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.get_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.get_issue",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","issue_number":7}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["title"] != "bug" {
		t.Errorf("title = %v, want bug", data["title"])
	}
}

func TestGetIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.get_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","issue_number":1}`},
		{"missing repo", `{"owner":"o","issue_number":1}`},
		{"missing issue_number", `{"owner":"o","repo":"r"}`},
		{"zero issue_number", `{"owner":"o","repo":"r","issue_number":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.get_issue",
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
