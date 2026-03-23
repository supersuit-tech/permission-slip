package microsoft

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
		if r.URL.Path != "/me/events" {
			t.Errorf("expected path /me/events, got %s", r.URL.Path)
		}

		var body graphEventRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Subject != "Team Meeting" {
			t.Errorf("expected subject 'Team Meeting', got %q", body.Subject)
		}
		if body.Start.TimeZone != "America/New_York" {
			t.Errorf("expected timezone 'America/New_York', got %q", body.Start.TimeZone)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "event-1",
			"subject": "Team Meeting",
			"start":   map[string]string{"dateTime": "2024-01-15T09:00:00", "timeZone": "America/New_York"},
			"end":     map[string]string{"dateTime": "2024-01-15T10:00:00", "timeZone": "America/New_York"},
			"webLink": "https://outlook.office365.com/calendar/item/event-1",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		Subject:  "Team Meeting",
		Start:    "2024-01-15T09:00:00",
		End:      "2024-01-15T10:00:00",
		TimeZone: "America/New_York",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
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
	if data["id"] != "event-1" {
		t.Errorf("expected id 'event-1', got %q", data["id"])
	}
	if data["subject"] != "Team Meeting" {
		t.Errorf("expected subject 'Team Meeting', got %q", data["subject"])
	}
}

func TestCreateCalendarEvent_WithCalendarID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/calendars/cal-xyz/events" {
			t.Errorf("expected calendar-scoped path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "event-cal",
			"subject": "X",
			"start":   map[string]string{"dateTime": "2024-01-15T09:00:00", "timeZone": "UTC"},
			"end":     map[string]string{"dateTime": "2024-01-15T10:00:00", "timeZone": "UTC"},
			"webLink": "https://example.com",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		CalendarID: "cal-xyz",
		Subject:    "X",
		Start:      "2024-01-15T09:00:00",
		End:        "2024-01-15T10:00:00",
		TimeZone:   "UTC",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCalendarEvent_WithAttendees(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphEventRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Attendees) != 2 {
			t.Errorf("expected 2 attendees, got %d", len(body.Attendees))
		}
		if body.Location == nil || body.Location.DisplayName != "Conference Room A" {
			t.Errorf("expected location 'Conference Room A'")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "event-2",
			"subject": "Meeting",
			"start":   map[string]string{"dateTime": "2024-01-15T09:00:00", "timeZone": "UTC"},
			"end":     map[string]string{"dateTime": "2024-01-15T10:00:00", "timeZone": "UTC"},
			"webLink": "https://outlook.office365.com/calendar/item/event-2",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(createCalendarEventParams{
		Subject:   "Meeting",
		Start:     "2024-01-15T09:00:00",
		End:       "2024-01-15T10:00:00",
		Attendees: []string{"alice@example.com", "bob@example.com"},
		Location:  "Conference Room A",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCalendarEvent_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"start": "2024-01-15T09:00:00",
		"end":   "2024-01-15T10:00:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_MissingStart(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject": "Meeting",
		"end":     "2024-01-15T10:00:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing start")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_InvalidAttendeeEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"subject":   "Meeting",
		"start":     "2024-01-15T09:00:00",
		"end":       "2024-01-15T10:00:00",
		"attendees": []string{"not-an-email"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid attendee email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCalendarEvent_DefaultTimeZone(t *testing.T) {
	t.Parallel()

	var params createCalendarEventParams
	params.defaults()
	if params.TimeZone != "UTC" {
		t.Errorf("expected default timezone 'UTC', got %q", params.TimeZone)
	}
}

func TestCreateCalendarEvent_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCalendarEventAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_calendar_event",
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
