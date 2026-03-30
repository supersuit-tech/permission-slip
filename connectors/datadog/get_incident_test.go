package datadog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetIncident_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v2/incidents/inc-abc-123" {
			t.Errorf("path = %s, want /api/v2/incidents/inc-abc-123", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":   "inc-abc-123",
				"type": "incidents",
				"attributes": map[string]any{
					"title":  "Database latency spike",
					"status": "active",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_incident"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_incident",
		Parameters:  json.RawMessage(`{"incident_id":"inc-abc-123"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestGetIncident_MissingIncidentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.get_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_incident",
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

func TestGetIncident_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []string{"Incident not found"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.get_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.get_incident",
		Parameters:  json.RawMessage(`{"incident_id":"nonexistent"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
