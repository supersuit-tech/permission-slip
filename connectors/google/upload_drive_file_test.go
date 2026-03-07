package google

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUploadDriveFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/upload/drive/v3/files" {
			t.Errorf("expected path /upload/drive/v3/files, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		// Verify multipart content.
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			t.Fatalf("failed to parse Content-Type: %v", err)
		}
		if !strings.HasPrefix(mediaType, "multipart/") {
			t.Fatalf("expected multipart content type, got %s", mediaType)
		}

		reader := multipart.NewReader(r.Body, params["boundary"])

		// Part 1: metadata.
		part, err := reader.NextPart()
		if err != nil {
			t.Fatalf("failed to read metadata part: %v", err)
		}
		var meta driveUploadMetadata
		if err := json.NewDecoder(part).Decode(&meta); err != nil {
			t.Fatalf("failed to decode metadata: %v", err)
		}
		if meta.Name != "test.txt" {
			t.Errorf("expected name 'test.txt', got %q", meta.Name)
		}

		// Part 2: content.
		part, err = reader.NextPart()
		if err != nil {
			t.Fatalf("failed to read content part: %v", err)
		}
		content, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("failed to read content: %v", err)
		}
		if string(content) != "Hello, Drive!" {
			t.Errorf("expected content 'Hello, Drive!', got %q", string(content))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveUploadResponse{
			ID:   "new-file-123",
			Name: "test.txt",
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		Name:    "test.txt",
		Content: "Hello, Drive!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
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
	if data["id"] != "new-file-123" {
		t.Errorf("expected id 'new-file-123', got %q", data["id"])
	}
	if data["name"] != "test.txt" {
		t.Errorf("expected name 'test.txt', got %q", data["name"])
	}
}

func TestUploadDriveFile_WithFolder(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		_, params, _ := mime.ParseMediaType(contentType)
		reader := multipart.NewReader(r.Body, params["boundary"])

		part, _ := reader.NextPart()
		var meta driveUploadMetadata
		json.NewDecoder(part).Decode(&meta)

		if len(meta.Parents) != 1 || meta.Parents[0] != "folder-abc" {
			t.Errorf("expected parent folder-abc, got %v", meta.Parents)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveUploadResponse{ID: "file-1", Name: "doc.txt"})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		Name:     "doc.txt",
		Content:  "Content here",
		FolderID: "folder-abc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadDriveFile_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"content": "hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
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

func TestUploadDriveFile_MissingContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"name": "test.txt"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadDriveFile_ContentTooLarge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	// Create content larger than maxUploadBytes (4 MB).
	largeContent := strings.Repeat("x", maxUploadBytes+1)
	params, _ := json.Marshal(uploadDriveFileParams{
		Name:    "large.txt",
		Content: largeContent,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadDriveFile_InvalidFolderID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	invalidIDs := []string{"../etc", "folder with spaces", "folder'inject"}
	for _, id := range invalidIDs {
		params, _ := json.Marshal(uploadDriveFileParams{
			Name:     "test.txt",
			Content:  "hello",
			FolderID: id,
		})
		_, err := action.Execute(t.Context(), connectors.ActionRequest{
			ActionType:  "google.upload_drive_file",
			Parameters:  params,
			Credentials: validCreds(),
		})
		if err == nil {
			t.Fatalf("expected error for invalid folder_id %q", id)
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("expected ValidationError for %q, got: %T", id, err)
		}
	}
}

func TestUploadDriveFile_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		Name:    "test.txt",
		Content: "hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
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

func TestUploadDriveFile_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		Name:    "test.txt",
		Content: "hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestUploadDriveFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.upload_drive_file",
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
