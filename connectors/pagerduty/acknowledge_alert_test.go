package pagerduty

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAcknowledgeAlert_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/incidents/P1234567" {
			t.Errorf("path = %s, want /incidents/P1234567", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		incident := reqBody["incident"].(map[string]any)
		if incident["status"] != "acknowledged" {
			t.Errorf("status = %v, want acknowledged", incident["status"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{
				"id":     "P1234567",
				"status": "acknowledged",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.acknowledge_alert"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.acknowledge_alert",
		Parameters:  json.RawMessage(`{"incident_id":"P1234567"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestAcknowledgeAlert_MissingIncidentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.acknowledge_alert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.acknowledge_alert",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAcknowledgeAlert_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.acknowledge_alert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.acknowledge_alert",
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
