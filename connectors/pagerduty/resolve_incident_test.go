package pagerduty

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestResolveIncident_Success(t *testing.T) {
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
		if incident["status"] != "resolved" {
			t.Errorf("status = %v, want resolved", incident["status"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{
				"id":     "P1234567",
				"status": "resolved",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.resolve_incident"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.resolve_incident",
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

func TestResolveIncident_MissingIncidentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.resolve_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.resolve_incident",
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
