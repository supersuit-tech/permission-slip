package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteCalendarEvent_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/calendars/primary/events/evt-to-delete" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), calendarBaseURL: srv.URL}
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(deleteCalendarEventParams{EventID: "evt-to-delete"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["event_id"] != "evt-to-delete" {
		t.Errorf("expected event_id 'evt-to-delete', got %q", data["event_id"])
	}
	if data["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %q", data["status"])
	}
}

func TestDeleteCalendarEvent_CustomCalendar(t *testing.T) {
	t.Parallel()

	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), calendarBaseURL: srv.URL}
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(deleteCalendarEventParams{EventID: "evt-xyz", CalendarID: "work@example.com"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// url.PathEscape does not encode '@' (valid path char), so we accept either form.
	if capturedPath != "/calendars/work@example.com/events/evt-xyz" &&
		capturedPath != "/calendars/work%40example.com/events/evt-xyz" {
		t.Errorf("unexpected path: %s", capturedPath)
	}
}

func TestDeleteCalendarEvent_MissingEventID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(deleteCalendarEventParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing event_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeleteCalendarEvent_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 404, "message": "Event not found"},
		})
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), calendarBaseURL: srv.URL}
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(deleteCalendarEventParams{EventID: "nonexistent"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestDeleteCalendarEvent_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteCalendarEventAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_calendar_event",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
