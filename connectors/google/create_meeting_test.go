package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateMeeting_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/calendars/primary/events" {
			t.Errorf("expected path /calendars/primary/events, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("conferenceDataVersion"); got != "1" {
			t.Errorf("expected conferenceDataVersion=1, got %q", got)
		}

		var body meetingEventRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Summary != "Team Standup" {
			t.Errorf("expected summary 'Team Standup', got %q", body.Summary)
		}
		if body.ConferenceData.CreateRequest.ConferenceSolutionKey.Type != "hangoutsMeet" {
			t.Errorf("expected conferenceSolutionKey type 'hangoutsMeet', got %q",
				body.ConferenceData.CreateRequest.ConferenceSolutionKey.Type)
		}
		if len(body.Attendees) != 1 {
			t.Errorf("expected 1 attendee, got %d", len(body.Attendees))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meetingEventResponse{
			ID:       "event-123",
			HTMLLink: "https://calendar.google.com/event?eid=123",
			Summary:  "Team Standup",
			Status:   "confirmed",
			ConferenceData: &meetingConferenceDataResponse{
				EntryPoints: []meetingEntryPoint{
					{EntryPointType: "video", URI: "https://meet.google.com/abc-defg-hij"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(createMeetingParams{
		Summary:   "Team Standup",
		StartTime: "2024-01-15T09:00:00-05:00",
		EndTime:   "2024-01-15T09:30:00-05:00",
		Attendees: []string{"colleague@example.com"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
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
	if data["meet_link"] != "https://meet.google.com/abc-defg-hij" {
		t.Errorf("expected meet_link, got %q", data["meet_link"])
	}
	if data["status"] != "confirmed" {
		t.Errorf("expected status 'confirmed', got %q", data["status"])
	}
}

func TestCreateMeeting_NoConferenceData(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meetingEventResponse{
			ID:       "event-456",
			HTMLLink: "https://calendar.google.com/event?eid=456",
			Status:   "confirmed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(createMeetingParams{
		Summary:   "Quick Sync",
		StartTime: "2024-01-15T14:00:00Z",
		EndTime:   "2024-01-15T14:30:00Z",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
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
	if data["id"] != "event-456" {
		t.Errorf("expected id 'event-456', got %q", data["id"])
	}
	if _, ok := data["meet_link"]; ok {
		t.Error("expected no meet_link when conferenceData is nil")
	}
}

func TestCreateMeeting_MissingSummary(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"start_time": "2024-01-15T09:00:00Z",
		"end_time":   "2024-01-15T10:00:00Z",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
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

func TestCreateMeeting_MissingStartTime(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"summary":  "Meeting",
		"end_time": "2024-01-15T10:00:00Z",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing start_time")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateMeeting_EndBeforeStart(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(createMeetingParams{
		Summary:   "Meeting",
		StartTime: "2024-01-15T10:00:00Z",
		EndTime:   "2024-01-15T09:00:00Z",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for end before start")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateMeeting_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createMeetingAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
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

func TestCreateMeeting_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    401,
				"message": "Invalid Credentials",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), "", srv.URL, "")
	action := &createMeetingAction{conn: conn}

	params, _ := json.Marshal(createMeetingParams{
		Summary:   "Meeting",
		StartTime: "2024-01-15T09:00:00Z",
		EndTime:   "2024-01-15T10:00:00Z",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}
