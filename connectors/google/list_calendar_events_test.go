package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListCalendarEvents_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/calendars/primary/events" {
			t.Errorf("expected path /calendars/primary/events, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendarListResponse{
			Items: []calendarListItem{
				{
					ID:      "event-1",
					Summary: "Team Standup",
					Status:  "confirmed",
					Start:   calendarListDateTime{DateTime: "2024-01-15T09:00:00-05:00"},
					End:     calendarListDateTime{DateTime: "2024-01-15T09:30:00-05:00"},
				},
				{
					ID:      "event-2",
					Summary: "All-day Review",
					Status:  "confirmed",
					Start:   calendarListDateTime{Date: "2024-01-16"},
					End:     calendarListDateTime{Date: "2024-01-17"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL)
	action := &listCalendarEventsAction{conn: conn}

	params, _ := json.Marshal(listCalendarEventsParams{MaxResults: 10})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendar_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Events []eventSummary `json:"events"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(data.Events))
	}
	if data.Events[0].Summary != "Team Standup" {
		t.Errorf("expected first event summary 'Team Standup', got %q", data.Events[0].Summary)
	}
	// All-day event should use Date field.
	if data.Events[1].StartTime != "2024-01-16" {
		t.Errorf("expected all-day start '2024-01-16', got %q", data.Events[1].StartTime)
	}
}

func TestListCalendarEvents_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendarListResponse{Items: nil})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL)
	action := &listCalendarEventsAction{conn: conn}

	params, _ := json.Marshal(listCalendarEventsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendar_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Events []eventSummary `json:"events"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Events) != 0 {
		t.Errorf("expected 0 events, got %d", len(data.Events))
	}
}

func TestListCalendarEvents_InvalidTimeMin(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCalendarEventsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"time_min": "not-a-date",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendar_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid time_min")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListCalendarEvents_TimeMaxBeforeTimeMin(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCalendarEventsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"time_min": "2024-01-15T10:00:00-05:00",
		"time_max": "2024-01-15T09:00:00-05:00",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendar_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for time_max before time_min")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListCalendarEvents_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL)
	action := &listCalendarEventsAction{conn: conn}

	params, _ := json.Marshal(listCalendarEventsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendar_events",
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

func TestListCalendarEvents_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCalendarEventsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendar_events",
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
