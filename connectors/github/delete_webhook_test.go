package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteWebhook_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/repos/o/r/hooks/123" {
			t.Errorf("path = %s, want /repos/o/r/hooks/123", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.delete_webhook"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.delete_webhook",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","hook_id":123}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestDeleteWebhook_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.delete_webhook"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","hook_id":1}`},
		{"missing repo", `{"owner":"o","hook_id":1}`},
		{"missing hook_id", `{"owner":"o","repo":"r"}`},
		{"zero hook_id", `{"owner":"o","repo":"r","hook_id":0}`},
		{"negative hook_id", `{"owner":"o","repo":"r","hook_id":-1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.delete_webhook",
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
