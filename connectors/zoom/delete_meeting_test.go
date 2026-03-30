package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteMeeting_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/meetings/123456" {
			t.Errorf("expected path /meetings/123456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteMeetingAction{conn: conn}

	params, _ := json.Marshal(deleteMeetingParams{MeetingID: "123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.delete_meeting",
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
	if data["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", data["deleted"])
	}
}

func TestDeleteMeeting_WithReminder(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("schedule_for_reminder") != "true" {
			t.Errorf("expected schedule_for_reminder=true, got %q", r.URL.Query().Get("schedule_for_reminder"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteMeetingAction{conn: conn}

	remind := true
	params, _ := json.Marshal(deleteMeetingParams{MeetingID: "123456", ScheduleForReminder: &remind})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.delete_meeting",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteMeeting_MissingMeetingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteMeetingAction{conn: conn}

	params, _ := json.Marshal(deleteMeetingParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.delete_meeting",
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

func TestDeleteMeeting_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(zoomAPIError{Code: 3001, Message: "Meeting does not exist."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteMeetingAction{conn: conn}

	params, _ := json.Marshal(deleteMeetingParams{MeetingID: "999999"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.delete_meeting",
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

func TestDeleteMeeting_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteMeetingAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.delete_meeting",
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
