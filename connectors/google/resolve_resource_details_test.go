package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		chatBaseURL:     srv.URL,
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

func TestResolveResourceDetails_DriveFile_SupportsAllDrives(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Shared receipt.pdf","mimeType":"application/pdf"}`))
	}))
	defer srv.Close()

	conn := &GoogleConnector{
		client:       srv.Client(),
		driveBaseURL: srv.URL,
	}
	params, _ := json.Marshal(map[string]string{"file_id": sharedDriveID})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.get_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["file_name"] != "Shared receipt.pdf" {
		t.Errorf("expected file_name, got %v", details["file_name"])
	}
	q, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	assertSupportsAllDrives(t, q)
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
		if details["presentation_title"] != "Q1 Review Deck" {
			t.Errorf("%s: expected presentation_title 'Q1 Review Deck', got %v", actionType, details["presentation_title"])
		}
	}
}

func TestResolveResourceDetails_ChatSpace_Invalid(t *testing.T) {
	conn := New()
	for _, spaceName := range []string{"AAA", "spaces/", "spaces/foo/bar", "spaces/..", "spaces/foo?bar"} {
		params, _ := json.Marshal(map[string]string{"space_name": spaceName})
		if _, err := conn.ResolveResourceDetails(context.Background(), "google.send_chat_message", params, validCreds()); err == nil {
			t.Errorf("expected error for space_name=%q", spaceName)
		}
	}
}

func TestResolveResourceDetails_ChatSpace(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/v1/spaces/": `{"displayName":"Dev Team"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"space_name": "spaces/AAQAohK4ZL0", "text": "Hello"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.send_chat_message", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["space_display_name"] != "Dev Team" {
		t.Errorf("expected space_display_name 'Dev Team', got %v", details["space_display_name"])
	}
}

func TestResolveResourceDetails_CalendarSummary(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/calendars/work@example.com": `{"summary":"Work Calendar"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"calendar_id": "work@example.com"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.list_calendar_events", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["calendar_name"] != "Work Calendar" {
		t.Errorf("expected calendar_name, got %v", details["calendar_name"])
	}
}

func TestResolveResourceDetails_CalendarSummary_DefaultPrimary(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/calendars/primary": `{"summary":"alice@example.com"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.list_calendar_events", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["calendar_name"] != "alice@example.com" {
		t.Errorf("expected calendar_name for primary, got %v", details["calendar_name"])
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

	details, err = conn.ResolveResourceDetails(context.Background(), "google.download_attachment", params, validCreds())
	if err != nil {
		t.Fatalf("download_attachment: unexpected error: %v", err)
	}
	if details["subject"] != "Weekly Update" {
		t.Errorf("download_attachment: expected subject 'Weekly Update', got %v", details["subject"])
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
		"/gmail/v1/users/me/threads/":  `{"id":"th1","messages":[{"id":"msg456","internalDate":"1","payload":{"mimeType":"text/plain","headers":[{"name":"Subject","value":"Re: Budget Discussion"}],"body":{"data":"SGk="}}}]}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"thread_id": "th1", "message_id": "msg456"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.send_email_reply", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["subject"] != "Re: Budget Discussion" {
		t.Errorf("expected subject, got %v", details["subject"])
	}
	et, ok := details["email_thread"].(map[string]any)
	if !ok {
		t.Fatalf("expected email_thread in details, got %T", details["email_thread"])
	}
	if et["subject"] != "Re: Budget Discussion" {
		t.Errorf("email_thread.subject: %v", et["subject"])
	}
}

func TestResolveResourceDetails_DriveFolder(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/drive/v3/files/": `{"name":"Receipts"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"folder_id": "1folderAbc"})
	for _, actionType := range []string{"google.upload_drive_file", "google.list_drive_files", "google.search_drive"} {
		details, err := conn.ResolveResourceDetails(context.Background(), actionType, params, validCreds())
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", actionType, err)
		}
		if details["folder_name"] != "Receipts" {
			t.Errorf("%s: expected folder_name 'Receipts', got %v", actionType, details["folder_name"])
		}
	}
}

func TestResolveResourceDetails_DriveFolder_DefaultMyDrive(t *testing.T) {
	conn := New()
	params, _ := json.Marshal(map[string]string{"name": "notes.md"})

	details, err := conn.ResolveResourceDetails(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["folder_name"] != "My Drive" {
		t.Errorf("expected folder_name 'My Drive', got %v", details["folder_name"])
	}

	details, err = conn.ResolveResourceDetails(context.Background(), "google.create_drive_folder", params, validCreds())
	if err != nil {
		t.Fatalf("create_drive_folder: unexpected error: %v", err)
	}
	if details["folder_name"] != "My Drive" {
		t.Errorf("create_drive_folder: expected folder_name 'My Drive', got %v", details["folder_name"])
	}
	if details["parent_name"] != "My Drive" {
		t.Errorf("create_drive_folder: expected parent_name 'My Drive', got %v", details["parent_name"])
	}
}

func TestResolveResourceDetails_ListDriveFiles_NoFolderID(t *testing.T) {
	conn := New()
	params, _ := json.Marshal(map[string]string{"query": "report"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.list_drive_files", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != nil {
		t.Errorf("expected nil details when folder_id is omitted, got %v", details)
	}
}

func TestResolveResourceDetails_CreateDriveFolder_Parent(t *testing.T) {
	srv, conn := testResolveServer(t, map[string]string{
		"/drive/v3/files/": `{"name":"Projects"}`,
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"name": "Q1", "parent_id": "1parentFolder"})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.create_drive_folder", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["folder_name"] != "Projects" {
		t.Errorf("expected folder_name 'Projects', got %v", details["folder_name"])
	}
	if details["parent_name"] != "Projects" {
		t.Errorf("expected parent_name 'Projects', got %v", details["parent_name"])
	}
}

func TestResolveResourceDetails_DriveFolder_SupportsAllDrives(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Shared Receipts"}`))
	}))
	defer srv.Close()

	conn := &GoogleConnector{
		client:       srv.Client(),
		driveBaseURL: srv.URL,
	}
	params, _ := json.Marshal(map[string]string{"folder_id": sharedDriveID})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["folder_name"] != "Shared Receipts" {
		t.Errorf("expected folder_name, got %v", details["folder_name"])
	}
	q, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	assertSupportsAllDrives(t, q)
}

func TestResolveResourceDetails_DriveFolder_SharedDriveFallback(t *testing.T) {
	var fileHits, driveHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case len(r.URL.Path) >= len("/drive/v3/files/") && r.URL.Path[:len("/drive/v3/files/")] == "/drive/v3/files/":
			fileHits++
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"message":"File not found","code":404}}`))
		case len(r.URL.Path) >= len("/drive/v3/drives/") && r.URL.Path[:len("/drive/v3/drives/")] == "/drive/v3/drives/":
			driveHits++
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Finance Shared Drive"}`))
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := &GoogleConnector{
		client:       srv.Client(),
		driveBaseURL: srv.URL,
	}
	params, _ := json.Marshal(map[string]string{"folder_id": sharedDriveID})
	details, err := conn.ResolveResourceDetails(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["folder_name"] != "Finance Shared Drive" {
		t.Errorf("expected shared drive name, got %v", details["folder_name"])
	}
	if fileHits != 1 || driveHits != 1 {
		t.Errorf("expected files.get then drives.get, got fileHits=%d driveHits=%d", fileHits, driveHits)
	}
}

func TestResolveResourceDetails_DriveFolder_InvalidID(t *testing.T) {
	conn := New()
	params, _ := json.Marshal(map[string]string{"folder_id": "not a valid id"})
	if _, err := conn.ResolveResourceDetails(context.Background(), "google.upload_drive_file", params, validCreds()); err == nil {
		t.Fatal("expected error for invalid folder_id")
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

	// Missing space_name
	_, err = conn.ResolveResourceDetails(context.Background(), "google.send_chat_message", params, validCreds())
	if err == nil {
		t.Error("expected error for missing space_name")
	}
}
