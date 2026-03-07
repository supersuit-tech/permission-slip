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

		// Verify the payload is nested correctly under meta.payload.
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		meta, ok := reqBody["meta"].(map[string]any)
		if !ok {
			t.Fatal("request body missing 'meta' object")
		}
		payload, ok := meta["payload"].(map[string]any)
		if !ok {
			t.Fatal("meta missing 'payload' object")
		}
		if payload["env"] != "production" {
			t.Errorf("payload.env = %v, want production", payload["env"])
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

func TestTriggerRunbook_EmptyPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		meta, ok := reqBody["meta"].(map[string]any)
		if !ok {
			t.Fatal("request body missing 'meta' object")
		}
		// When no payload is provided, meta should be empty (no null payload).
		if _, hasPayload := meta["payload"]; hasPayload {
			t.Error("meta should not contain 'payload' key when payload is empty")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "instance-789"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.trigger_runbook"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.trigger_runbook",
		Parameters:  json.RawMessage(`{"workflow_id":"wf-abc-123"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
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
