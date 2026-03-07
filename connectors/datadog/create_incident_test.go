package datadog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateIncident_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v2/incidents" {
			t.Errorf("path = %s, want /api/v2/incidents", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		data := reqBody["data"].(map[string]any)
		attrs := data["attributes"].(map[string]any)
		if attrs["title"] != "Database latency spike" {
			t.Errorf("title = %v, want %q", attrs["title"], "Database latency spike")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":   "incident-123",
				"type": "incidents",
				"attributes": map[string]any{
					"title": "Database latency spike",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.create_incident"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.create_incident",
		Parameters:  json.RawMessage(`{"title":"Database latency spike","severity":"SEV-2","customer_impacted":true}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestCreateIncident_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.create_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.create_incident",
		Parameters:  json.RawMessage(`{"severity":"SEV-2"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateIncident_InvalidSeverity(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.create_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.create_incident",
		Parameters:  json.RawMessage(`{"title":"Test","severity":"CRITICAL"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateIncident_DefaultSeverity(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		data := reqBody["data"].(map[string]any)
		attrs := data["attributes"].(map[string]any)
		fields := attrs["fields"].(map[string]any)
		severity := fields["severity"].(map[string]any)
		if severity["value"] != "UNKNOWN" {
			t.Errorf("severity = %v, want UNKNOWN", severity["value"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "inc-1"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.create_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.create_incident",
		Parameters:  json.RawMessage(`{"title":"Test incident"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
