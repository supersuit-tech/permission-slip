package google

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestConstraintMetadataActionSupport_SharedDriveActions(t *testing.T) {
	t.Parallel()
	c := New()

	for _, actionType := range []string{
		"google.upload_drive_file",
		"google.create_drive_folder",
		"google.list_drive_files",
		"google.search_drive",
		"google.get_drive_file",
		"google.sheets_read_range",
		"google.sheets_write_range",
		"google.sheets_append_rows",
		"google.sheets_list_sheets",
	} {
		fields, ok := c.ConstraintMetadataActionSupport(actionType)
		if !ok {
			t.Errorf("%s: expected meta constraint support", actionType)
			continue
		}
		if len(fields) != 1 || fields[0] != "drive_id" {
			t.Errorf("%s: expected [drive_id], got %v", actionType, fields)
		}
	}

	if _, ok := c.ConstraintMetadataActionSupport("google.send_email"); ok {
		t.Error("send_email should not advertise Drive $meta fields")
	}
	if _, ok := c.ConstraintMetadataActionSupport("google.delete_drive_file"); ok {
		t.Error("delete_drive_file should not advertise Shared Drive $meta fields")
	}
}

func TestResolveConstraintMetadata_NestedSharedDriveFolder(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case len(r.URL.Path) >= len("/drive/v3/files/") && r.URL.Path[:len("/drive/v3/files/")] == "/drive/v3/files/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"2026-639-receipts","driveId":"` + sharedDriveID + `","parents":["` + sharedDriveID + `"]}`))
		case len(r.URL.Path) >= len("/drive/v3/drives/") && r.URL.Path[:len("/drive/v3/drives/")] == "/drive/v3/drives/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Chiedo's Assistant Drive"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"folder_id": "1Xv2Naa6LjElcSK55wb9HigrLrAaYPE0d"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
}

func TestResolveConstraintMetadata_SharedDriveRoot(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case len(r.URL.Path) >= len("/drive/v3/files/") && r.URL.Path[:len("/drive/v3/files/")] == "/drive/v3/files/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Drive","driveId":"` + sharedDriveID + `"}`))
		case len(r.URL.Path) >= len("/drive/v3/drives/") && r.URL.Path[:len("/drive/v3/drives/")] == "/drive/v3/drives/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Assistant Drive"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"folder_id": sharedDriveID})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
}

func TestResolveConstraintMetadata_SharedDriveRootViaDrivesGet(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case len(r.URL.Path) >= len("/drive/v3/files/") && r.URL.Path[:len("/drive/v3/files/")] == "/drive/v3/files/":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"message":"File not found","code":404}}`))
		case len(r.URL.Path) >= len("/drive/v3/drives/") && r.URL.Path[:len("/drive/v3/drives/")] == "/drive/v3/drives/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Finance Shared Drive"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"parent_id": sharedDriveID})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.create_drive_folder", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
}

func TestResolveConstraintMetadata_MyDriveFolderOmitsDriveID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) < len("/drive/v3/files/") || r.URL.Path[:len("/drive/v3/files/")] != "/drive/v3/files/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Receipts"}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"folder_id": "1myDriveFolder"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if _, ok := meta["drive_id"]; ok {
		t.Errorf("My Drive folder should omit drive_id, got %v", meta["drive_id"])
	}
}

func TestResolveConstraintMetadata_OmittedFolderIsMyDriveRoot(t *testing.T) {
	t.Parallel()
	conn := New()
	params, _ := json.Marshal(map[string]string{"name": "receipt.pdf"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.upload_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if _, ok := meta["drive_id"]; ok {
		t.Errorf("My Drive root should omit drive_id, got %v", meta["drive_id"])
	}
}

func TestResolveConstraintMetadata_LookupFailureIsUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"backend error","code":500}}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"folder_id": "1someFolder"})
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.upload_drive_file", params, validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}

func TestResolveConstraintMetadata_UnsupportedAction(t *testing.T) {
	t.Parallel()
	conn := New()
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.send_email", json.RawMessage(`{}`), validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}

func TestResolveConstraintMetadata_InvalidFolderID(t *testing.T) {
	t.Parallel()
	conn := New()
	params, _ := json.Marshal(map[string]string{"folder_id": "../etc/passwd"})
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.upload_drive_file", params, validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}

func TestResolveConstraintMetadata_ListFolderOnSharedDrive(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case len(r.URL.Path) >= len("/drive/v3/files/") && r.URL.Path[:len("/drive/v3/files/")] == "/drive/v3/files/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"2026-639-receipts","driveId":"` + sharedDriveID + `","parents":["` + sharedDriveID + `"]}`))
		case len(r.URL.Path) >= len("/drive/v3/drives/") && r.URL.Path[:len("/drive/v3/drives/")] == "/drive/v3/drives/":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"name":"Assistant Drive"}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"folder_id": "1nestedFolder", "query": "receipt"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.list_drive_files", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
}

func TestResolveConstraintMetadata_ListDriveIDParamVerified(t *testing.T) {
	t.Parallel()
	var driveHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) < len("/drive/v3/drives/") || r.URL.Path[:len("/drive/v3/drives/")] != "/drive/v3/drives/" {
			t.Errorf("expected drives.get to verify drive_id, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		driveHits++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Assistant Drive"}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"drive_id": sharedDriveID})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.search_drive", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
	if driveHits != 1 {
		t.Errorf("drives.get hits = %d, want 1", driveHits)
	}
}

func TestResolveConstraintMetadata_ListAllFilesOmitsDriveID(t *testing.T) {
	t.Parallel()
	conn := New()
	params, _ := json.Marshal(map[string]string{"query": "report"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.list_drive_files", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if _, ok := meta["drive_id"]; ok {
		t.Errorf("unscoped list should omit drive_id, got %v", meta["drive_id"])
	}
}

func TestResolveConstraintMetadata_GetFileOnSharedDrive(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) < len("/drive/v3/files/") || r.URL.Path[:len("/drive/v3/files/")] != "/drive/v3/files/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"receipt.pdf","mimeType":"application/pdf","driveId":"` + sharedDriveID + `"}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"file_id": "1fileOnSharedDrive"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.get_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
}

func TestResolveConstraintMetadata_GetFileMyDriveOmitsDriveID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"notes.md","mimeType":"text/markdown"}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"file_id": "1myDriveFile"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.get_drive_file", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if _, ok := meta["drive_id"]; ok {
		t.Errorf("My Drive file should omit drive_id, got %v", meta["drive_id"])
	}
}

func TestResolveConstraintMetadata_SheetsOnSharedDrive(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) < len("/drive/v3/files/") || r.URL.Path[:len("/drive/v3/files/")] != "/drive/v3/files/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Budget","mimeType":"application/vnd.google-apps.spreadsheet","driveId":"` + sharedDriveID + `"}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"spreadsheet_id": "1sheetOnSharedDrive", "range": "Sheet1!A1:B2"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.sheets_read_range", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["drive_id"] != sharedDriveID {
		t.Errorf("drive_id = %v, want %s", meta["drive_id"], sharedDriveID)
	}
}

func TestResolveConstraintMetadata_ListDriveIDLookupFailure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"backend error","code":500}}`))
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"drive_id": sharedDriveID})
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.list_drive_files", params, validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}

func TestConstraintMetadataActionSupport_CalendarWrites(t *testing.T) {
	t.Parallel()
	c := New()

	for _, actionType := range []string{
		"google.create_calendar_event",
		"google.update_calendar_event",
		"google.delete_calendar_event",
		"google.create_meeting",
	} {
		fields, ok := c.ConstraintMetadataActionSupport(actionType)
		if !ok {
			t.Errorf("%s: expected meta constraint support", actionType)
			continue
		}
		if len(fields) != 1 || fields[0] != "calendar_id" {
			t.Errorf("%s: expected [calendar_id], got %v", actionType, fields)
		}
	}

	if _, ok := c.ConstraintMetadataActionSupport("google.send_email"); ok {
		t.Error("send_email should not advertise Calendar $meta fields")
	}
	if _, ok := c.ConstraintMetadataActionSupport("google.list_calendar_events"); ok {
		t.Error("list_calendar_events should not advertise Calendar write $meta fields")
	}
}

func TestResolveConstraintMetadata_CanonicalCalendarID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendars/work@example.com" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"work@example.com","summary":"Work Calendar"}`))
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"calendar_id": "work@example.com"})
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.create_calendar_event", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["calendar_id"] != "work@example.com" {
		t.Errorf("calendar_id = %v, want work@example.com", meta["calendar_id"])
	}
}

func TestResolveConstraintMetadata_PrimaryAliasCanonicalizes(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendars/primary" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"alice@example.com","summary":"Alice"}`))
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	for _, params := range []json.RawMessage{
		json.RawMessage(`{}`),
		json.RawMessage(`{"calendar_id":"primary"}`),
		json.RawMessage(`{"calendar_id":"","summary":"Standup"}`),
	} {
		meta, err := conn.ResolveConstraintMetadata(context.Background(), "google.create_meeting", params, validCreds())
		if err != nil {
			t.Fatalf("params %s: %v", params, err)
		}
		if meta["calendar_id"] != "alice@example.com" {
			t.Errorf("params %s: calendar_id = %v, want alice@example.com", params, meta["calendar_id"])
		}
	}
}

func TestResolveConstraintMetadata_CalendarLookupFailureIsUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"backend error","code":500}}`))
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"calendar_id": "work@example.com"})
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.update_calendar_event", params, validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}

func TestResolveConstraintMetadata_MissingCanonicalIDIsUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"summary":"Work Calendar"}`))
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	params, _ := json.Marshal(map[string]string{"calendar_id": "work@example.com"})
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.delete_calendar_event", params, validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}

func TestResolveConstraintMetadata_InvalidCalendarID(t *testing.T) {
	t.Parallel()
	conn := New()
	params, _ := json.Marshal(map[string]string{"calendar_id": "../etc/passwd"})
	_, err := conn.ResolveConstraintMetadata(context.Background(), "google.create_calendar_event", params, validCreds())
	if !errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
		t.Fatalf("expected ErrConstraintMetadataUnavailable, got %v", err)
	}
}
