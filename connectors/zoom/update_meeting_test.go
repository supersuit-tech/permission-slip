package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateMeeting_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/meetings/123456" {
			t.Errorf("expected path /meetings/123456, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body zoomUpdateMeetingRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Topic != "Updated Standup" {
			t.Errorf("expected topic 'Updated Standup', got %q", body.Topic)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateMeetingAction{conn: conn}

	params, _ := json.Marshal(updateMeetingParams{
		MeetingID: "123456",
		Topic:     "Updated Standup",
		Duration:  45,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.update_meeting",
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
	if data["updated"] != true {
		t.Errorf("expected updated=true, got %v", data["updated"])
	}
	if data["meeting_id"] != "123456" {
		t.Errorf("expected meeting_id '123456', got %v", data["meeting_id"])
	}
}

func TestUpdateMeeting_MissingMeetingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateMeetingAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"topic": "No ID"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.update_meeting",
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

func TestUpdateMeeting_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(zoomAPIError{Code: 3001, Message: "Meeting does not exist."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateMeetingAction{conn: conn}

	params, _ := json.Marshal(updateMeetingParams{MeetingID: "999999", Topic: "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.update_meeting",
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

func TestUpdateMeeting_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateMeetingAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.update_meeting",
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
