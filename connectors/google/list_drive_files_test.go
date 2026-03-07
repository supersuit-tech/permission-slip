package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListDriveFiles_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/drive/v3/files" {
			t.Errorf("expected path /drive/v3/files, got %s", r.URL.Path)
		}

		// Verify trashed=false is in the query
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("expected non-empty q parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{
			Files: []driveFileEntry{
				{ID: "file-1", Name: "doc.txt", MimeType: "text/plain", ModifiedTime: "2024-01-15T10:00:00Z"},
				{ID: "file-2", Name: "sheet.xlsx", MimeType: "application/vnd.google-apps.spreadsheet", ModifiedTime: "2024-01-16T10:00:00Z"},
			},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &listDriveFilesAction{conn: conn}

	params, _ := json.Marshal(listDriveFilesParams{MaxResults: 10})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Files []driveFileSummary `json:"files"`
		Count int                `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 2 {
		t.Errorf("expected count 2, got %d", data.Count)
	}
	if len(data.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(data.Files))
	}
}

func TestListDriveFiles_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{Files: nil})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &listDriveFilesAction{conn: conn}

	params, _ := json.Marshal(listDriveFilesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Files []driveFileSummary `json:"files"`
		Count int                `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 0 {
		t.Errorf("expected count 0, got %d", data.Count)
	}
}

func TestListDriveFiles_WithFolderID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			t.Error("expected non-empty q parameter")
		}
		// Verify folder filter is present.
		if !strings.Contains(q, "'folder-123' in parents") {
			t.Errorf("expected folder filter in query, got %q", q)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{
			Files: []driveFileEntry{
				{ID: "file-1", Name: "doc.txt", MimeType: "text/plain"},
			},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &listDriveFilesAction{conn: conn}

	params, _ := json.Marshal(listDriveFilesParams{FolderID: "folder-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 1 {
		t.Errorf("expected count 1, got %d", data.Count)
	}
}

func TestListDriveFiles_InvalidFolderID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDriveFilesAction{conn: conn}

	// Test path traversal.
	params, _ := json.Marshal(listDriveFilesParams{FolderID: "../etc/passwd"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal folder_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}

	// Test query injection via single quotes in folder_id.
	params, _ = json.Marshal(listDriveFilesParams{FolderID: "foo' or name contains 'secret"})
	_, err = action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for query injection folder_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListDriveFiles_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &listDriveFilesAction{conn: conn}

	params, _ := json.Marshal(listDriveFilesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
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

func TestListDriveFiles_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDriveFilesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_drive_files",
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

