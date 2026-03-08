package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteCalendarEvent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedPath := "/calendars/primary/events/evt123"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"event_id": "evt123",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(result.Data, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out["event_id"] != "evt123" {
		t.Errorf("expected event_id evt123, got %s", out["event_id"])
	}
	if out["calendar_id"] != "primary" {
		t.Errorf("expected calendar_id primary, got %s", out["calendar_id"])
	}
	if out["status"] != "deleted" {
		t.Errorf("expected status deleted, got %s", out["status"])
	}
}

func TestDeleteCalendarEvent_CustomCalendar(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"event_id":    "evt456",
		"calendar_id": "work@example.com",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// url.PathEscape may or may not encode @; accept either form
	if gotPath != "/calendars/work@example.com/events/evt456" &&
		gotPath != "/calendars/work%40example.com/events/evt456" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	var out map[string]string
	if err := json.Unmarshal(result.Data, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out["calendar_id"] != "work@example.com" {
		t.Errorf("expected calendar_id work@example.com, got %s", out["calendar_id"])
	}
}

func TestDeleteCalendarEvent_MissingEventID(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing event_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDeleteCalendarEvent_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":404,"message":"Event not found"}}`))
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &deleteCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"event_id": "nonexistent",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestDeleteCalendarEvent_InvalidJSON(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &deleteCalendarEventAction{conn: conn}

	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  []byte(`{bad json`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
