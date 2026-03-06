package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListDriveFiles_Success(t *testing.T) {
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
			"value": []map[string]any{
				{
					"id":                   "file-1",
					"name":                 "report.docx",
					"size":                 1024,
					"webUrl":               "https://onedrive.live.com/file-1",
					"createdDateTime":      "2024-01-15T09:00:00Z",
					"lastModifiedDateTime": "2024-01-16T10:00:00Z",
					"file": map[string]string{
						"mimeType": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
					},
				},
				{
					"id":                   "folder-1",
					"name":                 "Documents",
					"size":                 0,
					"createdDateTime":      "2024-01-10T08:00:00Z",
					"lastModifiedDateTime": "2024-01-14T12:00:00Z",
					"folder": map[string]any{
						"childCount": 5,
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDriveFilesAction{conn: conn}

	params, _ := json.Marshal(listDriveFilesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []driveFileSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 items, got %d", len(summaries))
	}

	// Check file
	if summaries[0].ID != "file-1" {
		t.Errorf("expected id 'file-1', got %q", summaries[0].ID)
	}
	if summaries[0].Type != "file" {
		t.Errorf("expected type 'file', got %q", summaries[0].Type)
	}
	if summaries[0].MimeType == "" {
		t.Error("expected non-empty mime_type for file")
	}

	// Check folder
	if summaries[1].ID != "folder-1" {
		t.Errorf("expected id 'folder-1', got %q", summaries[1].ID)
	}
	if summaries[1].Type != "folder" {
		t.Errorf("expected type 'folder', got %q", summaries[1].Type)
	}
	if summaries[1].ChildCount != 5 {
		t.Errorf("expected child_count 5, got %d", summaries[1].ChildCount)
	}
}

func TestListDriveFiles_WithFolderPath(t *testing.T) {
	t.Parallel()

	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDriveFilesAction{conn: conn}

	params, _ := json.Marshal(listDriveFilesParams{FolderPath: "Documents/Work"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_drive_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestPath != "/me/drive/root:/Documents/Work:/children" {
		t.Errorf("expected path with folder, got %q", requestPath)
	}
}

func TestListDriveFiles_DefaultParams(t *testing.T) {
	t.Parallel()

	var params listDriveFilesParams
	params.defaults()
	if params.Top != 10 {
		t.Errorf("expected default top 10, got %d", params.Top)
	}
}

func TestListDriveFiles_TopClamped(t *testing.T) {
	t.Parallel()

	params := listDriveFilesParams{Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListDriveFiles_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDriveFilesAction{conn: conn}

	cases := []struct {
		name       string
		folderPath string
	}{
		{"dot-dot", "../../admin"},
		{"backslash", "Documents\\secret"},
		{"absolute", "/root/files"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, _ := json.Marshal(listDriveFilesParams{FolderPath: tc.folderPath})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "microsoft.list_drive_files",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error for invalid folder_path")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestListDriveFiles_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDriveFilesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_drive_files",
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
