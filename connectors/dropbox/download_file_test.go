package dropbox

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDownloadFile_Success(t *testing.T) {
	t.Parallel()

	fileContent := "hello world file content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sl.test-dropbox-token-123" {
			t.Errorf("bad auth header: %s", got)
		}

		var apiArg downloadAPIArg
		if err := json.Unmarshal([]byte(r.Header.Get("Dropbox-API-Arg")), &apiArg); err != nil {
			t.Fatalf("failed to parse Dropbox-API-Arg: %v", err)
		}
		if apiArg.Path != "/Documents/report.pdf" {
			t.Errorf("expected path /Documents/report.pdf, got %s", apiArg.Path)
		}

		metadata, _ := json.Marshal(downloadResultHeader{
			Name:        "report.pdf",
			PathDisplay: "/Documents/report.pdf",
			ID:          "id:abc123",
			Size:        int64(len(fileContent)),
		})
		w.Header().Set("Dropbox-API-Result", string(metadata))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte(fileContent))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &downloadFileAction{conn: conn}

	params, _ := json.Marshal(downloadFileParams{
		Path: "/Documents/report.pdf",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.download_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["name"] != "report.pdf" {
		t.Errorf("expected name report.pdf, got %v", data["name"])
	}

	decoded, err := base64.StdEncoding.DecodeString(data["content"].(string))
	if err != nil {
		t.Fatalf("failed to decode content: %v", err)
	}
	if string(decoded) != fileContent {
		t.Errorf("expected content %q, got %q", fileContent, string(decoded))
	}
}

func TestDownloadFile_MissingPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &downloadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.download_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDownloadFile_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error_summary": "path/not_found/",
			"error":         map[string]string{".tag": "path"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &downloadFileAction{conn: conn}

	params, _ := json.Marshal(downloadFileParams{Path: "/nonexistent.txt"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.download_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
