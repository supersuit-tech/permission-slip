package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchDrive_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/drive/v3/files" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{
			Files: []driveFileEntry{
				{ID: "file-1", Name: "Q1 Report", MimeType: "application/vnd.google-apps.document"},
				{ID: "file-2", Name: "Q1 Budget", MimeType: "application/vnd.google-apps.spreadsheet"},
			},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &searchDriveAction{conn: conn}

	params, _ := json.Marshal(searchDriveParams{Query: "Q1"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["count"].(float64) != 2 {
		t.Errorf("expected count 2, got %v", data["count"])
	}
}

func TestSearchDrive_ByFileType(t *testing.T) {
	t.Parallel()

	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{Files: []driveFileEntry{}})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &searchDriveAction{conn: conn}

	params, _ := json.Marshal(searchDriveParams{FileType: "document"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedQuery == "" {
		t.Error("expected query to be set")
	}
	// Verify it filters by document MIME type
	expected := "application/vnd.google-apps.document"
	if !containsStr(capturedQuery, expected) {
		t.Errorf("query %q should contain MIME type %q", capturedQuery, expected)
	}
}

func TestSearchDrive_NoFiltersProvided(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchDriveAction{conn: conn}

	params, _ := json.Marshal(searchDriveParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no filters provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchDrive_InvalidFileType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchDriveAction{conn: conn}

	params, _ := json.Marshal(searchDriveParams{FileType: "invalid-type"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid file_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchDrive_InvalidFolderID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchDriveAction{conn: conn}

	params, _ := json.Marshal(searchDriveParams{FolderID: "folder/../../etc"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid folder_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchDrive_MaxResultsClamp(t *testing.T) {
	t.Parallel()

	var capturedPageSize string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPageSize = r.URL.Query().Get("pageSize")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{Files: []driveFileEntry{}})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &searchDriveAction{conn: conn}

	params, _ := json.Marshal(searchDriveParams{Query: "test", MaxResults: 999})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPageSize != "100" {
		t.Errorf("expected pageSize clamped to 100, got %q", capturedPageSize)
	}
}

func TestSearchDrive_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchDriveAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.search_drive",
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

// containsStr is a helper to check substring containment in tests.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && searchContains(s, substr))
}

func searchContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
