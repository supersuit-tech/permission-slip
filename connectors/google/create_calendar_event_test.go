package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCalendarEvent_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/calendars/primary/events" {
			t.Errorf("expected path /calendars/primary/events, got %s", r.URL.Path)
		}

		var body calendarEventRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Summary != "Team Meeting" {
			t.Errorf("expected summary 'Team Meeting', got %q", body.Summary)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendarEventResponse{
			ID:       "event-123",
			HTMLLink: "https://calendar.google.com/event?eid=event-123",
			Status:   "confirmed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		Summary:   "Team Meeting",
		StartTime: "2024-01-15T09:00:00-05:00",
		EndTime:   "2024-01-15T10:00:00-05:00",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
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
	if data["id"] != "event-123" {
		t.Errorf("expected id 'event-123', got %q", data["id"])
	}
	if data["status"] != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", data["status"])
	}
}

func TestCreateCalendarEvent_WithAttendees(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body calendarEventRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Attendees) != 2 {
			t.Errorf("expected 2 attendees, got %d", len(body.Attendees))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendarEventResponse{
			ID:     "event-456",
			Status: "confirmed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		Summary:   "Team Sync",
		StartTime: "2024-01-15T09:00:00-05:00",
		EndTime:   "2024-01-15T10:00:00-05:00",
		Attendees: []string{"alice@example.com", "bob@example.com"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCalendarEvent_MissingSummary(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"start_time": "2024-01-15T09:00:00-05:00",
		"end_time":   "2024-01-15T10:00:00-05:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing summary")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_InvalidStartTime(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"summary":    "Meeting",
		"start_time": "not-a-date",
		"end_time":   "2024-01-15T10:00:00-05:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid start_time")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_InvalidEndTime(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"summary":    "Meeting",
		"start_time": "2024-01-15T09:00:00-05:00",
		"end_time":   "not-a-date",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid end_time")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_EndBeforeStart(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"summary":    "Meeting",
		"start_time": "2024-01-15T10:00:00-05:00",
		"end_time":   "2024-01-15T09:00:00-05:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
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

func TestCreateCalendarEvent_EndEqualsStart(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"summary":    "Meeting",
		"start_time": "2024-01-15T09:00:00-05:00",
		"end_time":   "2024-01-15T09:00:00-05:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for end_time equal to start_time")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		Summary:   "Meeting",
		StartTime: "2024-01-15T09:00:00-05:00",
		EndTime:   "2024-01-15T10:00:00-05:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
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

func TestCreateCalendarEvent_CalendarIDURLEncoded(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Calendar ID with @ should be decoded to the correct path by the server.
		wantPath := "/calendars/user@group.calendar.google.com/events"
		if r.URL.Path != wantPath {
			t.Errorf("expected path %q, got %q", wantPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendarEventResponse{
			ID:     "event-789",
			Status: "confirmed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		Summary:    "Meeting",
		StartTime:  "2024-01-15T09:00:00-05:00",
		EndTime:    "2024-01-15T10:00:00-05:00",
		CalendarID: "user@group.calendar.google.com",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCalendarEvent_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_calendar_event",
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
