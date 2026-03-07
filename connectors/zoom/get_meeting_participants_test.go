package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetMeetingParticipants_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/past_meetings/123456/participants" {
			t.Errorf("expected path /past_meetings/123456/participants, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomParticipantsResponse{
			Participants: []zoomParticipant{
				{
					ID:        "p-1",
					Name:      "Alice Smith",
					Email:     "alice@example.com",
					JoinTime:  "2024-01-15T09:00:00Z",
					LeaveTime: "2024-01-15T09:30:00Z",
					Duration:  1800,
				},
				{
					ID:        "p-2",
					Name:      "Bob Jones",
					Email:     "bob@example.com",
					JoinTime:  "2024-01-15T09:05:00Z",
					LeaveTime: "2024-01-15T09:30:00Z",
					Duration:  1500,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getMeetingParticipantsAction{conn: conn}

	params, _ := json.Marshal(getMeetingParticipantsParams{MeetingID: "123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting_participants",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalParticipants int               `json:"total_participants"`
		Participants      []participantItem `json:"participants"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalParticipants != 2 {
		t.Errorf("expected total_participants 2, got %d", data.TotalParticipants)
	}
	if len(data.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(data.Participants))
	}
	if data.Participants[0].Name != "Alice Smith" {
		t.Errorf("expected first participant 'Alice Smith', got %q", data.Participants[0].Name)
	}
	if data.Participants[0].Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got %q", data.Participants[0].Email)
	}
	if data.Participants[0].Duration != 1800 {
		t.Errorf("expected duration 1800, got %d", data.Participants[0].Duration)
	}
}

func TestGetMeetingParticipants_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomParticipantsResponse{Participants: nil})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getMeetingParticipantsAction{conn: conn}

	params, _ := json.Marshal(getMeetingParticipantsParams{MeetingID: "123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting_participants",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalParticipants int               `json:"total_participants"`
		Participants      []participantItem `json:"participants"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalParticipants != 0 {
		t.Errorf("expected total_participants 0, got %d", data.TotalParticipants)
	}
	if len(data.Participants) != 0 {
		t.Errorf("expected 0 participants, got %d", len(data.Participants))
	}
}

func TestGetMeetingParticipants_MissingMeetingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getMeetingParticipantsAction{conn: conn}

	params, _ := json.Marshal(getMeetingParticipantsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting_participants",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing meeting_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetMeetingParticipants_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(zoomAPIError{Code: 3001, Message: "Meeting does not exist."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getMeetingParticipantsAction{conn: conn}

	params, _ := json.Marshal(getMeetingParticipantsParams{MeetingID: "999999"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting_participants",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got: %T", err)
	}
}

func TestGetMeetingParticipants_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getMeetingParticipantsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting_participants",
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
