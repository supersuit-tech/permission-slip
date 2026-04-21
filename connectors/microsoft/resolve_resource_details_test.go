package microsoft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func testResolveServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *MicrosoftConnector) {
	t.Helper()
	srv := httptest.NewServer(handler)
	conn := newForTest(srv.Client(), srv.URL)
	return srv, conn
}

func TestResolveResourceDetails_DriveItem(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/drive/items/item123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"report.txt"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"item_id": "item123"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.get_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["file_name"] != "report.txt" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_DocumentTitle(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Notes.docx"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"item_id": "doc1"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.update_document", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["document_title"] != "Notes.docx" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_PresentationTitle(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Deck.pptx"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"item_id": "pres1"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.get_presentation", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["presentation_title"] != "Deck.pptx" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_WorkbookTitle(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Book.xlsx"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"item_id": "wb1"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.excel_read_range", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["workbook_title"] != "Book.xlsx" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_CalendarName_Default(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/calendar" {
			t.Errorf("expected /me/calendar, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Calendar"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.list_calendar_events", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["calendar_name"] != "Calendar" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_CalendarName_Specific(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/calendars/cal-abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"Work"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"calendar_id": "cal-abc"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.list_calendar_events", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["calendar_name"] != "Work" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_TeamName(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"Engineering"}`))
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"team_id": "team1"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.list_channels", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["team_name"] != "Engineering" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_TeamAndChannel(t *testing.T) {
	srv, conn := testResolveServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/teams/t1":
			_, _ = w.Write([]byte(`{"displayName":"Sales"}`))
		case "/teams/t1/channels/ch1":
			_, _ = w.Write([]byte(`{"displayName":"General"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer srv.Close()

	params, _ := json.Marshal(map[string]string{"team_id": "t1", "channel_id": "ch1"})
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.send_channel_message", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details["team_name"] != "Sales" || details["channel_name"] != "General" {
		t.Fatalf("got %#v", details)
	}
}

func TestResolveResourceDetails_UnknownAction(t *testing.T) {
	conn := New()
	details, err := conn.ResolveResourceDetails(context.Background(), "microsoft.send_email", []byte(`{}`), validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if details != nil {
		t.Fatalf("expected nil, got %#v", details)
	}
}

func TestResolveResourceDetails_MissingItemID(t *testing.T) {
	conn := New()
	_, err := conn.ResolveResourceDetails(context.Background(), "microsoft.get_drive_file", []byte(`{}`), validCreds())
	if err == nil {
		t.Fatal("expected error")
	}
}
