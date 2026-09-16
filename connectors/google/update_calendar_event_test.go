package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateCalendarEvent_Success(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":       "evt123",
			"summary":  "Updated Meeting",
			"status":   "confirmed",
			"htmlLink": "https://calendar.google.com/event?eid=evt123",
			"updated":  "2024-01-15T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id": "evt123",
		"summary":  "Updated Meeting",
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
	if out["id"] != "evt123" {
		t.Errorf("expected id evt123, got %s", out["id"])
	}
	if gotBody["summary"] != "Updated Meeting" {
		t.Errorf("expected summary in body, got %v", gotBody["summary"])
	}
}

func TestUpdateCalendarEvent_ClearAttendees(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "evt123",
			"status": "confirmed",
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id":        "evt123",
		"clear_attendees": true,
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	attendees, ok := gotBody["attendees"]
	if !ok {
		t.Fatal("expected attendees key in body")
	}
	arr, ok := attendees.([]any)
	if !ok || len(arr) != 0 {
		t.Errorf("expected empty attendees array, got %v", attendees)
	}
}

func TestUpdateCalendarEvent_ClearAttendeesAndAttendeesConflict(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id":        "evt123",
		"clear_attendees": true,
		"attendees":       []string{"user@example.com"},
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for conflicting clear_attendees and attendees")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateCalendarEvent_MissingEventID(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"summary": "Updated",
	})
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

func TestUpdateCalendarEvent_NoFieldsProvided(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id": "evt123",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no update fields provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateCalendarEvent_MismatchedTimes(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id":   "evt123",
		"start_time": "2024-01-15T10:00:00Z",
		// end_time missing
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for mismatched start/end times")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateCalendarEvent_InvalidJSON(t *testing.T) {
	conn := newCalendarForTest(nil, "http://unused")
	action := &updateCalendarEventAction{conn: conn}

	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  []byte(`{bad json`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestUpdateCalendarEvent_InstanceScope(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "abc123_20260120T150000Z",
			"status": "confirmed",
		})
	}))
	defer srv.Close()

	action := &updateCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]any{
		"event_id":   "abc123_20260120T150000Z",
		"scope":      "instance",
		"start_time": "2026-01-21T16:00:00Z",
		"end_time":   "2026-01-21T17:00:00Z",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/calendars/primary/events/abc123_20260120T150000Z" {
		t.Errorf("expected instance PATCH path, got %s", gotPath)
	}
	var out map[string]string
	json.Unmarshal(result.Data, &out)
	if out["scope"] != "instance" {
		t.Errorf("expected scope in result, got %v", out)
	}
}

func TestUpdateCalendarEvent_SeriesRecurrence(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "abc123",
			"status": "confirmed",
		})
	}))
	defer srv.Close()

	action := &updateCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]any{
		"event_id":   "abc123",
		"scope":      "series",
		"recurrence": []string{"RRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=TU"},
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/calendars/primary/events/abc123" {
		t.Errorf("expected master PATCH path, got %s", gotPath)
	}
	rec, _ := gotBody["recurrence"].([]any)
	if len(rec) != 1 || rec[0] != "RRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=TU" {
		t.Errorf("expected recurrence in body, got %v", gotBody["recurrence"])
	}
}

func TestUpdateCalendarEvent_InstanceScopeRejectsMasterID(t *testing.T) {
	action := &updateCalendarEventAction{conn: newCalendarForTest(nil, "http://unused")}
	params, _ := json.Marshal(map[string]any{
		"event_id": "abc123",
		"scope":    "instance",
		"summary":  "Nope",
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

func TestUpdateCalendarEvent_RecurrenceRejectedOnInstance(t *testing.T) {
	action := &updateCalendarEventAction{conn: newCalendarForTest(nil, "http://unused")}
	params, _ := json.Marshal(map[string]any{
		"event_id":   "abc123_20260120T150000Z",
		"scope":      "instance",
		"recurrence": []string{"RRULE:FREQ=WEEKLY;BYDAY=TU"},
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

func TestUpdateCalendarEvent_ThisAndFollowingSplitsSeries(t *testing.T) {
	var patchRecurrence []any
	var insertBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123_20260120T150000Z":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:               "abc123_20260120T150000Z",
				RecurringEventID: "abc123",
				Summary:          "Weekly standup",
				Start:            calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
				End:              calendarEventDateTime{DateTime: "2026-01-20T16:00:00Z"},
				OriginalStart:    calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:         "abc123",
				Summary:    "Weekly standup",
				Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU"},
				Start:      calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
				End:        calendarEventDateTime{DateTime: "2026-01-06T16:00:00Z"},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/calendars/primary/events/abc123":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			patchRecurrence, _ = body["recurrence"].([]any)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"id": "abc123", "status": "confirmed"})
		case r.Method == http.MethodPost && r.URL.Path == "/calendars/primary/events":
			json.NewDecoder(r.Body).Decode(&insertBody)
			json.NewEncoder(w).Encode(map[string]string{
				"id":       "newseries",
				"status":   "confirmed",
				"htmlLink": "https://calendar.google.com/event?eid=newseries",
				"updated":  "2026-01-15T10:00:00Z",
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	action := &updateCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]any{
		"event_id":   "abc123_20260120T150000Z",
		"scope":      "this_and_following",
		"start_time": "2026-01-20T16:00:00Z",
		"end_time":   "2026-01-20T17:00:00Z",
	})
	result, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patchRecurrence) != 1 {
		t.Fatalf("expected master recurrence PATCH, got %v", patchRecurrence)
	}
	rec, _ := patchRecurrence[0].(string)
	if rec != "RRULE:FREQ=WEEKLY;BYDAY=TU;UNTIL=20260120T145959Z" {
		t.Errorf("unexpected truncated RRULE %q", rec)
	}
	if insertBody["summary"] != "Weekly standup" {
		t.Errorf("new series should copy summary, got %v", insertBody["summary"])
	}
	start, _ := insertBody["start"].(map[string]any)
	if start["dateTime"] != "2026-01-20T16:00:00Z" {
		t.Errorf("new series should use updated start, got %v", insertBody["start"])
	}
	var out map[string]string
	json.Unmarshal(result.Data, &out)
	if out["id"] != "newseries" {
		t.Errorf("expected new series id, got %v", out)
	}
	if out["original_event_id"] != "abc123" {
		t.Errorf("expected original_event_id, got %v", out)
	}
	if out["scope"] != "this_and_following" {
		t.Errorf("expected scope, got %v", out)
	}
}

func TestUpdateCalendarEvent_InstanceViaMasterAndStart(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/calendars/primary/events/abc123/instances"):
			json.NewEncoder(w).Encode(calendarInstancesResponse{
				Items: []calendarEventResource{{
					ID:               "abc123_20260120T150000Z",
					RecurringEventID: "abc123",
					Start:            calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
					OriginalStart:    calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
				}},
			})
		case r.Method == http.MethodPatch:
			gotPath = r.URL.Path
			json.NewEncoder(w).Encode(map[string]string{"id": "abc123_20260120T150000Z", "status": "confirmed"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	action := &updateCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]any{
		"event_id":       "abc123",
		"scope":          "instance",
		"instance_start": "2026-01-20T15:00:00Z",
		"summary":        "Moved",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/calendars/primary/events/abc123_20260120T150000Z" {
		t.Errorf("expected instance PATCH, got %s", gotPath)
	}
}

func TestUpdateCalendarEvent_ThisAndFollowingFirstInstancePatchesMaster(t *testing.T) {
	var patched bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123_20260106T150000Z":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:               "abc123_20260106T150000Z",
				RecurringEventID: "abc123",
				Start:            calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
				End:              calendarEventDateTime{DateTime: "2026-01-06T16:00:00Z"},
				OriginalStart:    calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/calendars/primary/events/abc123":
			json.NewEncoder(w).Encode(calendarEventResource{
				ID:         "abc123",
				Summary:    "Weekly standup",
				Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU"},
				Start:      calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
				End:        calendarEventDateTime{DateTime: "2026-01-06T16:00:00Z"},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/calendars/primary/events/abc123":
			patched = true
			json.NewEncoder(w).Encode(map[string]string{"id": "abc123", "summary": "Renamed", "status": "confirmed"})
		case r.Method == http.MethodPost:
			t.Error("should not create a new series when editing the first instance")
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	action := &updateCalendarEventAction{conn: newCalendarForTest(srv.Client(), srv.URL)}
	params, _ := json.Marshal(map[string]any{
		"event_id": "abc123_20260106T150000Z",
		"scope":    "this_and_following",
		"summary":  "Renamed",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Fatal("expected master PATCH")
	}
}

func TestUpdateCalendarEvent_ReplaceReminders(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "evt123",
			"status": "confirmed",
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id":         "evt123",
		"reminder_minutes": []int{1, 5},
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reminders, ok := gotBody["reminders"].(map[string]any)
	if !ok {
		t.Fatalf("expected reminders object, got %v", gotBody["reminders"])
	}
	if reminders["useDefault"] != false {
		t.Errorf("expected useDefault false, got %v", reminders["useDefault"])
	}
	overrides, ok := reminders["overrides"].([]any)
	if !ok || len(overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %v", reminders["overrides"])
	}
}

func TestUpdateCalendarEvent_ResetRemindersToDefault(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":     "evt123",
			"status": "confirmed",
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &updateCalendarEventAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"event_id": "evt123",
		"reminders": map[string]any{
			"use_default": true,
		},
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reminders, ok := gotBody["reminders"].(map[string]any)
	if !ok {
		t.Fatalf("expected reminders object, got %v", gotBody["reminders"])
	}
	if reminders["useDefault"] != true {
		t.Errorf("expected useDefault true, got %v", reminders["useDefault"])
	}
	if _, hasOverrides := reminders["overrides"]; hasOverrides {
		t.Errorf("expected overrides omitted when resetting to default, got %v", reminders["overrides"])
	}
}
