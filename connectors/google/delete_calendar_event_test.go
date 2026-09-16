package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
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

func TestDeleteCalendarEvent_InstanceScope(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	action := &deleteCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]string{
		"event_id": "abc123_20260120T150000Z",
		"scope":    "instance",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/calendars/primary/events/abc123_20260120T150000Z" {
		t.Errorf("expected instance DELETE path, got %s", gotPath)
	}
	var out map[string]string
	json.Unmarshal(result.Data, &out)
	if out["scope"] != "instance" || out["status"] != "deleted" {
		t.Errorf("unexpected result %v", out)
	}
}

func TestDeleteCalendarEvent_SeriesScope(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	action := &deleteCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]string{
		"event_id": "abc123",
		"scope":    "series",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/calendars/primary/events/abc123" {
		t.Errorf("expected master DELETE path, got %s", gotPath)
	}
}

func TestDeleteCalendarEvent_SeriesScopeRejectsInstanceID(t *testing.T) {
	action := &deleteCalendarEventAction{conn: newCalendarForTest(nil, "http://unused")}
	params, _ := json.Marshal(map[string]string{
		"event_id": "abc123_20260120T150000Z",
		"scope":    "series",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDeleteCalendarEvent_ThisAndFollowingTruncatesSeries(t *testing.T) {
	var patchRecurrence []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123_20260120T150000Z":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:               "abc123_20260120T150000Z",
				RecurringEventID: "abc123",
				Start:            calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
				End:              calendarEventDateTime{DateTime: "2026-01-20T16:00:00Z"},
				OriginalStart:    calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:         "abc123",
				Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU"},
				Start:      calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
				End:        calendarEventDateTime{DateTime: "2026-01-06T16:00:00Z"},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/calendars/primary/events/abc123":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			patchRecurrence, _ = body["recurrence"].([]any)
			json.NewEncoder(w).Encode(map[string]string{"id": "abc123", "status": "confirmed"})
		case r.Method == http.MethodDelete:
			t.Error("mid-series this_and_following should truncate, not DELETE")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	action := &deleteCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]string{
		"event_id": "abc123_20260120T150000Z",
		"scope":    "this_and_following",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patchRecurrence) != 1 {
		t.Fatalf("expected recurrence PATCH, got %v", patchRecurrence)
	}
	rec, _ := patchRecurrence[0].(string)
	if rec != "RRULE:FREQ=WEEKLY;BYDAY=TU;UNTIL=20260120T145959Z" {
		t.Errorf("unexpected truncated RRULE %q", rec)
	}
	var out map[string]string
	json.Unmarshal(result.Data, &out)
	if out["status"] != "truncated" {
		t.Errorf("expected truncated status, got %v", out)
	}
	if out["original_event_id"] != "abc123" {
		t.Errorf("expected original_event_id, got %v", out)
	}
}

func TestDeleteCalendarEvent_ThisAndFollowingFirstInstanceDeletesSeries(t *testing.T) {
	var deletedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123_20260106T150000Z":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:               "abc123_20260106T150000Z",
				RecurringEventID: "abc123",
				Start:            calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
				OriginalStart:    calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:         "abc123",
				Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU"},
				Start:      calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/calendars/primary/events/abc123":
			deletedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	action := &deleteCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]string{
		"event_id": "abc123_20260106T150000Z",
		"scope":    "this_and_following",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedPath != "/calendars/primary/events/abc123" {
		t.Errorf("expected master DELETE, got %s", deletedPath)
	}
	var out map[string]string
	json.Unmarshal(result.Data, &out)
	if out["status"] != "deleted" {
		t.Errorf("expected deleted status, got %v", out)
	}
}
