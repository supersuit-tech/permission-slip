package datadog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestTriggerRunbook_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v2/workflows/wf-abc-123/instances" {
			t.Errorf("path = %s, want /api/v2/workflows/wf-abc-123/instances", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":   "instance-456",
				"type": "workflow_instances",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.trigger_runbook"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.trigger_runbook",
		Parameters:  json.RawMessage(`{"workflow_id":"wf-abc-123","payload":{"env":"production"}}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestTriggerRunbook_MissingWorkflowID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.trigger_runbook"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.trigger_runbook",
		Parameters:  json.RawMessage(`{"payload":{"env":"production"}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestTriggerRunbook_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.trigger_runbook"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.trigger_runbook",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
