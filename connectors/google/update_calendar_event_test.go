package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
