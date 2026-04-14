package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRemoveLabel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		// Label name is URL-encoded per segment.
		if r.URL.EscapedPath() != "/repos/o/r/issues/5/labels/needs%20triage" {
			t.Errorf("escaped path = %s, want /repos/o/r/issues/5/labels/needs%%20triage", r.URL.EscapedPath())
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.remove_label"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.remove_label",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","issue_number":5,"name":"needs triage"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestRemoveLabel_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.remove_label"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing name", `{"owner":"o","repo":"r","issue_number":1}`},
		{"empty name", `{"owner":"o","repo":"r","issue_number":1,"name":""}`},
		{"zero issue_number", `{"owner":"o","repo":"r","issue_number":0,"name":"bug"}`},
		{"missing owner", `{"repo":"r","issue_number":1,"name":"bug"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.remove_label",
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
