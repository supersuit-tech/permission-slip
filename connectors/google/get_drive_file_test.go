package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetDriveFile_MetadataOnly(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/drive/v3/files/file-abc" {
			t.Errorf("expected path /drive/v3/files/file-abc, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveFileMetadata{
			ID:           "file-abc",
			Name:         "report.docx",
			MimeType:     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			ModifiedTime: "2024-01-15T10:00:00Z",
			Size:         "12345",
			WebViewLink:  "https://drive.google.com/file/d/file-abc/view",
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{FileID: "file-abc"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data driveFileResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "file-abc" {
		t.Errorf("expected id 'file-abc', got %q", data.ID)
	}
	if data.Name != "report.docx" {
		t.Errorf("expected name 'report.docx', got %q", data.Name)
	}
	if data.Content != "" {
		t.Error("expected empty content for metadata-only request")
	}
}

func TestGetDriveFile_WithGoogleDocsContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/drive/v3/files/doc-123":
			// Metadata request.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(driveFileMetadata{
				ID:       "doc-123",
				Name:     "My Document",
				MimeType: "application/vnd.google-apps.document",
			})
		case "/drive/v3/files/doc-123/export":
			// Export request.
			mimeType := r.URL.Query().Get("mimeType")
			if mimeType != "text/plain" {
				t.Errorf("expected export mimeType text/plain, got %q", mimeType)
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Hello, this is the document content."))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{FileID: "doc-123", IncludeContent: true})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data driveFileResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Content != "Hello, this is the document content." {
		t.Errorf("unexpected content: %q", data.Content)
	}
}

func TestGetDriveFile_WithTextFileContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files/txt-456" && r.URL.Query().Get("alt") == "media":
			// Download request.
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Plain text file content."))
		case r.URL.Path == "/drive/v3/files/txt-456":
			// Metadata request.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(driveFileMetadata{
				ID:       "txt-456",
				Name:     "notes.txt",
				MimeType: "text/plain",
				Size:     "24",
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{FileID: "txt-456", IncludeContent: true})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data driveFileResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Content != "Plain text file content." {
		t.Errorf("unexpected content: %q", data.Content)
	}
}

func TestGetDriveFile_BinaryFileNoContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/drive/v3/files/img-789" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveFileMetadata{
			ID:       "img-789",
			Name:     "photo.png",
			MimeType: "image/png",
			Size:     "1048576",
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{FileID: "img-789", IncludeContent: true})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data driveFileResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Content != "" {
		t.Error("expected empty content for binary file")
	}
	if data.ContentSkippedReason == "" {
		t.Error("expected content_skipped_reason for binary file with include_content=true")
	}
}

func TestGetDriveFile_MissingFileID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing file_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetDriveFile_InvalidFileID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{FileID: "../../etc/passwd"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid file_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetDriveFile_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "Insufficient permissions"},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{FileID: "file-abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
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

func TestGetDriveFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDriveFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_drive_file",
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
