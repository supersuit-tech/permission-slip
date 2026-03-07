package pagerduty

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddNote_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/incidents/P1234567/notes" {
			t.Errorf("path = %s, want /incidents/P1234567/notes", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		note := reqBody["note"].(map[string]any)
		if note["content"] != "Investigating the root cause" {
			t.Errorf("content = %v, want %q", note["content"], "Investigating the root cause")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"note": map[string]any{
				"id":      "P_NOTE_1",
				"content": "Investigating the root cause",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.add_note"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.add_note",
		Parameters:  json.RawMessage(`{"incident_id":"P1234567","content":"Investigating the root cause"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestAddNote_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.add_note"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing incident_id", params: `{"content":"Note"}`},
		{name: "missing content", params: `{"incident_id":"P1234567"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "pagerduty.add_note",
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
