package dropbox

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUploadFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sl.test-dropbox-token-123" {
			t.Errorf("bad auth header: %s", got)
		}
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("expected application/octet-stream, got %s", r.Header.Get("Content-Type"))
		}

		// Validate Dropbox-API-Arg header
		var apiArg uploadAPIArg
		if err := json.Unmarshal([]byte(r.Header.Get("Dropbox-API-Arg")), &apiArg); err != nil {
			t.Fatalf("failed to parse Dropbox-API-Arg: %v", err)
		}
		if apiArg.Path != "/Documents/report.pdf" {
			t.Errorf("expected path /Documents/report.pdf, got %s", apiArg.Path)
		}
		if apiArg.Mode != "add" {
			t.Errorf("expected mode add, got %s", apiArg.Mode)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadResponse{
			Name:        "report.pdf",
			PathDisplay: "/Documents/report.pdf",
			ID:          "id:abc123",
			Size:        1024,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &uploadFileAction{conn: conn}

	boolTrue := true
	params, _ := json.Marshal(uploadFileParams{
		Path:       "/Documents/report.pdf",
		Content:    base64.StdEncoding.EncodeToString([]byte("file content")),
		Autorename: &boolTrue,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.upload_file",
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
	if data["id"] != "id:abc123" {
		t.Errorf("expected id id:abc123, got %v", data["id"])
	}
}

func TestUploadFile_DefaultAutorename(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiArg uploadAPIArg
		json.Unmarshal([]byte(r.Header.Get("Dropbox-API-Arg")), &apiArg)
		if !apiArg.Autorename {
			t.Errorf("expected autorename to default to true, got false")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(uploadResponse{
			Name:        "test.txt",
			PathDisplay: "/test.txt",
			ID:          "id:def456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &uploadFileAction{conn: conn}

	// Omit autorename — should default to true
	params, _ := json.Marshal(map[string]string{
		"path":    "/test.txt",
		"content": base64.StdEncoding.EncodeToString([]byte("data")),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.upload_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadFile_MissingPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"content": base64.StdEncoding.EncodeToString([]byte("data")),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.upload_file",
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

func TestUploadFile_InvalidBase64(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"path":    "/test.txt",
		"content": "not-valid-base64!!!",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.upload_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadFile_InvalidMode(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"path":    "/test.txt",
		"content": base64.StdEncoding.EncodeToString([]byte("data")),
		"mode":    "invalid",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.upload_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadFile_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error_summary": "path/conflict/file",
			"error":         map[string]string{".tag": "path"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(uploadFileParams{
		Path:    "/test.txt",
		Content: base64.StdEncoding.EncodeToString([]byte("data")),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.upload_file",
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
