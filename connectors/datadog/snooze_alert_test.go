package datadog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSnoozeAlert_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/monitor/12345/mute" {
			t.Errorf("path = %s, want /api/v1/monitor/12345/mute", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":    12345,
			"name":  "CPU Monitor",
			"state": map[string]any{"muted": true},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.snooze_alert"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.snooze_alert",
		Parameters:  json.RawMessage(`{"monitor_id":12345,"end":1700010000}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestSnoozeAlert_WithScopeAndEnd(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["scope"] != "host:web-1" {
			t.Errorf("scope = %v, want host:web-1", reqBody["scope"])
		}
		if reqBody["end"] != float64(1700010000) {
			t.Errorf("end = %v, want 1700010000", reqBody["end"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"id": 12345})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["datadog.snooze_alert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.snooze_alert",
		Parameters:  json.RawMessage(`{"monitor_id":12345,"end":1700010000,"scope":"host:web-1"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSnoozeAlert_NegativeMonitorID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.snooze_alert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.snooze_alert",
		Parameters:  json.RawMessage(`{"monitor_id":-1}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSnoozeAlert_MissingMonitorID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.snooze_alert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.snooze_alert",
		Parameters:  json.RawMessage(`{"end":1700010000}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSnoozeAlert_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["datadog.snooze_alert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "datadog.snooze_alert",
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
