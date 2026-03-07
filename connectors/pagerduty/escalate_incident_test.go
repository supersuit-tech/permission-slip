package pagerduty

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestEscalateIncident_Success(t *testing.T) {
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
		if incident["escalation_level"] != float64(2) {
			t.Errorf("escalation_level = %v, want 2", incident["escalation_level"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{
				"id":               "P1234567",
				"escalation_level": 2,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.escalate_incident"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.escalate_incident",
		Parameters:  json.RawMessage(`{"incident_id":"P1234567","escalation_level":2}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestEscalateIncident_WithPolicy(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		incident := reqBody["incident"].(map[string]any)
		policy := incident["escalation_policy"].(map[string]any)
		if policy["id"] != "PPOLICY1" {
			t.Errorf("escalation_policy.id = %v, want PPOLICY1", policy["id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{"id": "P1234567"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.escalate_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.escalate_incident",
		Parameters:  json.RawMessage(`{"incident_id":"P1234567","escalation_level":3,"escalation_policy_id":"PPOLICY1"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestEscalateIncident_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.escalate_incident"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing incident_id", params: `{"escalation_level":2}`},
		{name: "missing escalation_level", params: `{"incident_id":"P1234567"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "pagerduty.escalate_incident",
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
