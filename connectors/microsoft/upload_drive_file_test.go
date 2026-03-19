package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUploadDriveFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("expected octet-stream content type, got %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "file content here" {
			t.Errorf("expected body 'file content here', got %q", string(body))
		}

		// Check conflict behavior query param
		if !strings.Contains(r.URL.RawQuery, "conflictBehavior=rename") {
			t.Errorf("expected conflictBehavior=rename in query, got %q", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":                   "new-file-1",
			"name":                 "report.txt",
			"size":                 17,
			"webUrl":               "https://onedrive.live.com/new-file-1",
			"createdDateTime":      "2024-01-15T09:00:00Z",
			"lastModifiedDateTime": "2024-01-15T09:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		FilePath: "Documents/report.txt",
		Content:  "file content here",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res uploadDriveFileResult
	if err := json.Unmarshal(result.Data, &res); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if res.ID != "new-file-1" {
		t.Errorf("expected id 'new-file-1', got %q", res.ID)
	}
	if res.Name != "report.txt" {
		t.Errorf("expected name 'report.txt', got %q", res.Name)
	}
}

func TestUploadDriveFile_ConflictBehaviorReplace(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "conflictBehavior=replace") {
			t.Errorf("expected conflictBehavior=replace, got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "file-1",
			"name": "report.txt",
			"size": 5,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		FilePath:         "report.txt",
		Content:          "hello",
		ConflictBehavior: "replace",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadDriveFile_MissingFilePath(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"content": "data"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing file_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadDriveFile_EmptyContentXlsx(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Empty .xlsx uploads should use the minimal XLSX template,
		// not 0 bytes, so the file is a valid workbook.
		if len(body) == 0 {
			t.Errorf("expected XLSX template body for empty .xlsx upload, got 0 bytes")
		}
		if len(body) != len(minimalXLSX) {
			t.Errorf("expected body length %d (XLSX template), got %d", len(minimalXLSX), len(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "new-workbook-1",
			"name": "report.xlsx",
			"size": len(minimalXLSX),
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"file_path": "Documents/report.xlsx"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res uploadDriveFileResult
	if err := json.Unmarshal(result.Data, &res); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if res.ID != "new-workbook-1" {
		t.Errorf("expected id 'new-workbook-1', got %q", res.ID)
	}
}

func TestUploadDriveFile_EmptyContentXlsxUpperCase(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if len(body) != len(minimalXLSX) {
			t.Errorf("expected XLSX template for .XLSX extension, got %d bytes", len(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "new-workbook-2",
			"name": "Report.XLSX",
			"size": len(minimalXLSX),
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"file_path": "Documents/Report.XLSX"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadDriveFile_EmptyContentNonXlsx(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		// Non-.xlsx files with empty content should still upload 0 bytes.
		if len(body) != 0 {
			t.Errorf("expected empty body for non-xlsx file, got %d bytes", len(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "new-file-1",
			"name": "notes.txt",
			"size": 0,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"file_path": "Documents/notes.txt"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadDriveFile_ContentTooLarge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	bigContent := strings.Repeat("x", maxUploadContentSize+1)
	params, _ := json.Marshal(uploadDriveFileParams{
		FilePath: "big.txt",
		Content:  bigContent,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
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

func TestUploadDriveFile_InvalidConflictBehavior(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	params, _ := json.Marshal(uploadDriveFileParams{
		FilePath:         "test.txt",
		Content:          "data",
		ConflictBehavior: "invalid",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid conflict_behavior")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadDriveFile_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	cases := []struct {
		name     string
		filePath string
	}{
		{"dot-dot", "../../etc/passwd"},
		{"backslash", "Documents\\file.txt"},
		{"absolute", "/root/file.txt"},
		{"query-injection", "Documents/file.txt?$filter=malicious"},
		{"fragment-injection", "Documents/file.txt#fragment"},
		{"percent-encoding", "Documents%2F..%2Fetc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, _ := json.Marshal(uploadDriveFileParams{
				FilePath: tc.filePath,
				Content:  "data",
			})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "microsoft.upload_drive_file",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error for invalid file_path")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestUploadDriveFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadDriveFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.upload_drive_file",
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
