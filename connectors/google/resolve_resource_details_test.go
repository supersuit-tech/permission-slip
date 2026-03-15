package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testResolveServer creates an httptest.Server that routes requests by path prefix
// and returns the corresponding JSON body. Returns the server and a GoogleConnector
// pointed at it.
func testResolveServer(t *testing.T, routes map[string]string) (*httptest.Server, *GoogleConnector) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for prefix, body := range routes {
			if len(r.URL.Path) >= len(prefix) && r.URL.Path[:len(prefix)] == prefix {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(body))
				return
			}
		}
		t.Errorf("unexpected request path: %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	conn := &GoogleConnector{
		client:          srv.Client(),
		gmailBaseURL:    srv.URL,
		calendarBaseURL: srv.URL,
		slidesBaseURL:   srv.URL,
		sheetsBaseURL:   srv.URL,
		docsBaseURL:     srv.URL,
		driveBaseURL:    srv.URL,
	}
	return srv, conn
}

func TestResolveResourceDetails_CalendarEvent(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/calendars/": `{"summary":"Q1 Planning","start":{"dateTime":"2026-03-15T14:00:00Z"},"end":{"dateTime":"2026-03-15T15:00:00Z"}}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"event_id": "evt123", "calendar_id": "primary"})

	for _, actionType := range []string{"google.delete_calendar_event", "google.update_calendar_event"} {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["title"] != "Q1 Planning" {
			t.Errorf("%s: expected title 'Q1 Planning', got %v", actionType, details["title"])
		}
		if details["start_time"] != "2026-03-15T14:00:00Z" {
			t.Errorf("%s: expected start_time, got %v", actionType, details["start_time"])
		}
		if details["end_time"] != "2026-03-15T15:00:00Z" {
			t.Errorf("%s: expected end_time, got %v", actionType, details["end_time"])
		}
	}
}

func TestResolveResourceDetails_CalendarEvent_AllDayEvent(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/calendars/": `{"summary":"Company Holiday","start":{"date":"2026-12-25"},"end":{"date":"2026-12-26"}}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"event_id": "evt_allday"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.delete_calendar_event", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["start_time"] != "2026-12-25" {
		t.Errorf("expected date-only start_time, got %v", details["start_time"])
	}
}

func TestResolveResourceDetails_DriveFile(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/drive/v3/files/": `{"name":"Budget 2026.xlsx","mimeType":"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"file_id": "f123"})

	for _, actionType := range []string{"google.delete_drive_file", "google.get_drive_file"} {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["file_name"] != "Budget 2026.xlsx" {
			t.Errorf("%s: expected file_name, got %v", actionType, details["file_name"])
		}
		if details["mime_type"] == nil {
			t.Errorf("%s: expected mime_type", actionType)
		}
	}
}

func TestResolveResourceDetails_Document(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/v1/documents/": `{"title":"Project Spec"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"document_id": "doc123"})

	for _, actionType := range []string{"google.get_document", "google.update_document"} {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["title"] != "Project Spec" {
			t.Errorf("%s: expected title 'Project Spec', got %v", actionType, details["title"])
		}
	}
}

func TestResolveResourceDetails_Spreadsheet(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/spreadsheets/": `{"properties":{"title":"Budget Tracker"}}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"spreadsheet_id": "s123", "range": "Sheet1!A1:B5"})

	for _, actionType := range []string{"google.sheets_read_range", "google.sheets_write_range", "google.sheets_append_rows", "google.sheets_list_sheets"} {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["title"] != "Budget Tracker" {
			t.Errorf("%s: expected title 'Budget Tracker', got %v", actionType, details["title"])
		}
		// sheets_list_sheets doesn't have a range param
		if actionType != "google.sheets_list_sheets" {
			if details["range"] != "Sheet1!A1:B5" {
				t.Errorf("%s: expected range, got %v", actionType, details["range"])
			}
		}
	}
}

func TestResolveResourceDetails_Presentation(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/v1/presentations/": `{"title":"Q1 Review Deck"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"presentation_id": "p123"})

	for _, actionType := range []string{"google.get_presentation", "google.add_slide"} {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["title"] != "Q1 Review Deck" {
			t.Errorf("%s: expected title 'Q1 Review Deck', got %v", actionType, details["title"])
		}
	}
}

func TestResolveResourceDetails_Email(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/gmail/v1/users/me/messages/": `{"payload":{"headers":[{"name":"Subject","value":"Weekly Update"},{"name":"From","value":"alice@example.com"}]}}`,
		"/gmail/v1/users/me/threads/":  `{"messages":[{"id":"msg_from_thread"}]}`,
	})
	defer srv.Close()

	// read_email uses message_id
	params, _ := json.Marshal(map[string]string{"message_id": "msg123"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.read_email", params, validCreds())
	if err != nil {
		t.Fatalf("read_email: unexpected error: %v", err)
	}
	if details["subject"] != "Weekly Update" {
		t.Errorf("read_email: expected subject 'Weekly Update', got %v", details["subject"])
	}
	if details["from"] != "alice@example.com" {
		t.Errorf("read_email: expected from 'alice@example.com', got %v", details["from"])
	}

	// archive_email uses thread_id — fetches thread first, then message
	params, _ = json.Marshal(map[string]string{"thread_id": "thread123"})
	details, err = conn.ResolveResourceDetails(context.Background(), "google.archive_email", params, validCreds())
	if err != nil {
		t.Fatalf("archive_email: unexpected error: %v", err)
	}
	if details["subject"] != "Weekly Update" {
		t.Errorf("archive_email: expected subject, got %v", details["subject"])
	}
}

func TestResolveResourceDetails_EmailReply(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/gmail/v1/users/me/messages/": `{"payload":{"headers":[{"name":"Subject","value":"Re: Budget Discussion"}]}}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"message_id": "msg456"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.send_email_reply", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["subject"] != "Re: Budget Discussion" {
		t.Errorf("expected subject, got %v", details["subject"])
	}
}

func TestResolveResourceDetails_UnknownAction(t *testing.T) {
	conn := New()
	details, err := conn.ResolveResourceDetails(context.Background(), "google.unknown_action", []byte(`{}`), validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details for unknown action, got %v", details)
	}
}

func TestResolveResourceDetails_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"Not Found","code":404}}`))
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"event_id": "deleted_event"})
	_, err := conn.ResolveResourceDetails(context.Background(), "google.delete_calendar_event", params, validCreds())
	if err == nil {
		t.Fatal("expected error for 404 API response")
	}
}

func TestResolveResourceDetails_MissingParams(t *testing.T) {
	conn := New()

	// Missing event_id
	params, _ := json.Marshal(map[string]string{})
	_, err := conn.ResolveResourceDetails(context.Background(), "google.delete_calendar_event", params, validCreds())
	if err == nil {
		t.Error("expected error for missing event_id")
	}

	// Missing file_id
	_, err = conn.ResolveResourceDetails(context.Background(), "google.delete_drive_file", params, validCreds())
	if err == nil {
		t.Error("expected error for missing file_id")
	}

	// Missing document_id
	_, err = conn.ResolveResourceDetails(context.Background(), "google.get_document", params, validCreds())
	if err == nil {
		t.Error("expected error for missing document_id")
	}

	// Missing spreadsheet_id
	_, err = conn.ResolveResourceDetails(context.Background(), "google.sheets_read_range", params, validCreds())
	if err == nil {
		t.Error("expected error for missing spreadsheet_id")
	}

	// Missing presentation_id
	_, err = conn.ResolveResourceDetails(context.Background(), "google.get_presentation", params, validCreds())
	if err == nil {
		t.Error("expected error for missing presentation_id")
	}
}
