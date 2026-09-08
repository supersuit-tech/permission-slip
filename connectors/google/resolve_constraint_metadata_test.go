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

func TestConstraintMetadataActionSupport_DriveWrites(t *testing.T) {
	t.Parallel()
	c := New()

	for _, actionType := range []string{"google.upload_drive_file", "google.create_drive_folder"} {
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
	if _, ok := c.ConstraintMetadataActionSupport("google.list_drive_files"); ok {
		t.Error("list_drive_files should not advertise Drive write $meta fields")
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
