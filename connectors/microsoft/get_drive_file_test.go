package microsoft

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
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":                   "file-123",
			"name":                 "notes.txt",
			"size":                 256,
			"webUrl":               "https://onedrive.live.com/file-123",
			"createdDateTime":      "2024-01-15T09:00:00Z",
			"lastModifiedDateTime": "2024-01-16T10:00:00Z",
			"file": map[string]string{
				"mimeType": "text/plain",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{ItemID: "file-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail driveFileDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if detail.ID != "file-123" {
		t.Errorf("expected id 'file-123', got %q", detail.ID)
	}
	if detail.Name != "notes.txt" {
		t.Errorf("expected name 'notes.txt', got %q", detail.Name)
	}
	if detail.Type != "file" {
		t.Errorf("expected type 'file', got %q", detail.Type)
	}
	if detail.Content != "" {
		t.Errorf("expected no content, got %q", detail.Content)
	}
}

func TestGetDriveFile_WithContent(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// Metadata request
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"id":   "file-123",
				"name": "notes.txt",
				"size": 13,
				"file": map[string]string{
					"mimeType": "text/plain",
				},
			})
			return
		}
		// Content download request
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello, World!"))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{ItemID: "file-123", IncludeContent: true})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail driveFileDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if detail.Content != "Hello, World!" {
		t.Errorf("expected content 'Hello, World!', got %q", detail.Content)
	}
}

func TestGetDriveFile_BinaryContentRejected(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "file-123",
			"name": "photo.png",
			"size": 1024,
			"file": map[string]string{
				"mimeType": "image/png",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{ItemID: "file-123", IncludeContent: true})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for binary file content download")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetDriveFile_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetDriveFile_InvalidItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{ItemID: "../../../etc/passwd"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetDriveFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDriveFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
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

func TestGetDriveFile_FolderMetadata(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "folder-1",
			"name": "Documents",
			"size": 0,
			"folder": map[string]any{
				"childCount": 3,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{ItemID: "folder-1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail driveFileDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if detail.Type != "folder" {
		t.Errorf("expected type 'folder', got %q", detail.Type)
	}
}

func TestGetDriveFile_FolderContentSkipped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "folder-1",
			"name": "Documents",
			"size": 0,
			"folder": map[string]any{
				"childCount": 3,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDriveFileAction{conn: conn}

	params, _ := json.Marshal(getDriveFileParams{ItemID: "folder-1", IncludeContent: true})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail driveFileDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if detail.ContentSkipped == "" {
		t.Error("expected content_skipped to be set for folder with include_content=true")
	}
}

func TestValidateItemID(t *testing.T) {
	t.Parallel()

	valid := []string{"abc123", "ABC-123-DEF", "item!123", "item.with.dots"}
	for _, id := range valid {
		if err := validateItemID(id); err != nil {
			t.Errorf("expected valid item_id %q, got error: %v", id, err)
		}
	}

	invalid := []string{"../etc", "a/b", "a\\b", "a..b"}
	for _, id := range invalid {
		if err := validateItemID(id); err == nil {
			t.Errorf("expected invalid item_id %q to be rejected", id)
		}
	}
}
