package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateDriveFolder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/drive/v3/files" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body driveFolderCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.MimeType != "application/vnd.google-apps.folder" {
			t.Errorf("expected folder MIME type, got %q", body.MimeType)
		}
		if body.Name != "My Folder" {
			t.Errorf("expected name 'My Folder', got %q", body.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveFolderResponse{
			ID:          "folder-123",
			Name:        "My Folder",
			MimeType:    "application/vnd.google-apps.folder",
			WebViewLink: "https://drive.google.com/drive/folders/folder-123",
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &createDriveFolderAction{conn: conn}

	params, _ := json.Marshal(createDriveFolderParams{Name: "My Folder"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_drive_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "folder-123" {
		t.Errorf("expected id 'folder-123', got %q", data["id"])
	}
	if data["name"] != "My Folder" {
		t.Errorf("expected name 'My Folder', got %q", data["name"])
	}
}

func TestCreateDriveFolder_WithParent(t *testing.T) {
	t.Parallel()

	var capturedBody driveFolderCreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveFolderResponse{
			ID:   "folder-456",
			Name: "Sub Folder",
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &createDriveFolderAction{conn: conn}

	params, _ := json.Marshal(createDriveFolderParams{Name: "Sub Folder", ParentID: "parent-folder-id"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_drive_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedBody.Parents) != 1 || capturedBody.Parents[0] != "parent-folder-id" {
		t.Errorf("expected parents [parent-folder-id], got %v", capturedBody.Parents)
	}
}

func TestCreateDriveFolder_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDriveFolderAction{conn: conn}

	params, _ := json.Marshal(createDriveFolderParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_drive_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDriveFolder_InvalidParentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDriveFolderAction{conn: conn}

	params, _ := json.Marshal(createDriveFolderParams{Name: "folder", ParentID: "bad/path/../id"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_drive_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid parent_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDriveFolder_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "Insufficient permissions"},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &createDriveFolderAction{conn: conn}

	params, _ := json.Marshal(createDriveFolderParams{Name: "My Folder"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_drive_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCreateDriveFolder_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDriveFolderAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_drive_folder",
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
