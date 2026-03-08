package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateCalendarEvent_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/calendars/primary/events/evt-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendarEventResponse{
			ID:      "evt-abc",
			HTMLLink: "https://calendar.google.com/event?eid=abc",
			Status:  "confirmed",
			Updated: "2024-01-15T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), calendarBaseURL: srv.URL}
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(updateCalendarEventParams{
		EventID:   "evt-abc",
		Summary:   "Updated Meeting",
		StartTime: "2024-01-15T09:00:00-05:00",
		EndTime:   "2024-01-15T10:00:00-05:00",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
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
	if data["id"] != "evt-abc" {
		t.Errorf("expected id 'evt-abc', got %q", data["id"])
	}
}

func TestUpdateCalendarEvent_MissingEventID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(updateCalendarEventParams{Summary: "foo"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
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

func TestUpdateCalendarEvent_NoFieldsProvided(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(updateCalendarEventParams{EventID: "evt-abc"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no update fields provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateCalendarEvent_StartWithoutEnd(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(updateCalendarEventParams{
		EventID:   "evt-abc",
		StartTime: "2024-01-15T09:00:00-05:00",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when start_time provided without end_time")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateCalendarEvent_InvalidTimeRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(updateCalendarEventParams{
		EventID:   "evt-abc",
		StartTime: "2024-01-15T10:00:00-05:00",
		EndTime:   "2024-01-15T09:00:00-05:00",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for end_time before start_time")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateCalendarEvent_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid credentials"},
		})
	}))
	defer srv.Close()

	conn := &GoogleConnector{client: srv.Client(), calendarBaseURL: srv.URL}
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(updateCalendarEventParams{EventID: "evt-abc", Summary: "New title"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestUpdateCalendarEvent_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateCalendarEventAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_calendar_event",
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
