package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetMeeting_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/meetings/123456" {
			t.Errorf("expected path /meetings/123456, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomMeetingDetail{
			ID:        123456,
			UUID:      "uuid-123",
			Topic:     "Sprint Planning",
			Type:      2,
			StartTime: "2024-01-20T14:00:00Z",
			Duration:  60,
			Timezone:  "America/New_York",
			JoinURL:   "https://zoom.us/j/123456",
			StartURL:  "https://zoom.us/s/123456",
			Password:  "secret",
			Status:    "waiting",
			Settings: zoomMeetingSettings{
				JoinBeforeHost: true,
				WaitingRoom:    false,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getMeetingAction{conn: conn}

	params, _ := json.Marshal(getMeetingParams{MeetingID: "123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["topic"] != "Sprint Planning" {
		t.Errorf("expected topic 'Sprint Planning', got %v", data["topic"])
	}
	if data["join_url"] != "https://zoom.us/j/123456" {
		t.Errorf("expected join_url, got %v", data["join_url"])
	}
	settings, ok := data["settings"].(map[string]any)
	if !ok {
		t.Fatal("expected settings to be a map")
	}
	if settings["join_before_host"] != true {
		t.Errorf("expected join_before_host=true, got %v", settings["join_before_host"])
	}
}

func TestGetMeeting_URLEncodesSpecialChars(t *testing.T) {
	t.Parallel()

	// Zoom double-encoded UUIDs can contain '/' which must be path-escaped.
	const uuidWithSlash = "abc123==//def456"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The path should have the UUID properly escaped.
		expectedPath := "/meetings/abc123==%2F%2Fdef456"
		if r.URL.RawPath != expectedPath {
			t.Errorf("expected raw path %q, got %q", expectedPath, r.URL.RawPath)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomMeetingDetail{
			ID:    123456,
			UUID:  uuidWithSlash,
			Topic: "Test",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getMeetingAction{conn: conn}

	params, _ := json.Marshal(getMeetingParams{MeetingID: uuidWithSlash})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetMeeting_MissingMeetingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getMeetingAction{conn: conn}

	params, _ := json.Marshal(getMeetingParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting",
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

func TestGetMeeting_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(zoomAPIError{Code: 3001, Message: "Meeting does not exist."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getMeetingAction{conn: conn}

	params, _ := json.Marshal(getMeetingParams{MeetingID: "999999"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting",
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

func TestGetMeeting_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getMeetingAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_meeting",
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
