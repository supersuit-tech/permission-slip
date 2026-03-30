package slack

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUploadFile_Success(t *testing.T) {
	t.Parallel()

	// Track which endpoints were called.
	var getURLCalled, uploadCalled, completeCalled atomic.Bool

	// We need the upload URL to point back to our test server, but we don't
	// know the URL until the server starts. Use a mux that captures the
	// server URL at request time from the Host header.
	mux := http.NewServeMux()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/files.getUploadURLExternal", func(w http.ResponseWriter, r *http.Request) {
		getURLCalled.Store(true)
		var body getUploadURLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode getUploadURL body: %v", err)
			return
		}
		if body.Filename != "report.csv" {
			t.Errorf("expected filename 'report.csv', got %q", body.Filename)
		}
		if body.Length != 11 {
			t.Errorf("expected length 11, got %d", body.Length)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"upload_url": srv.URL + "/upload-target",
			"file_id":    "F123ABC",
		})
	})

	mux.HandleFunc("/upload-target", func(w http.ResponseWriter, r *http.Request) {
		uploadCalled.Store(true)
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data, got %q", ct)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("failed to parse multipart form: %v", err)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("failed to get form file: %v", err)
			return
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		if string(content) != "hello,world" {
			t.Errorf("expected file content 'hello,world', got %q", string(content))
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		completeCalled.Store(true)
		var body completeUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode complete body: %v", err)
			return
		}
		if len(body.Files) != 1 || body.Files[0].ID != "F123ABC" {
			t.Errorf("expected file ID 'F123ABC', got %v", body.Files)
		}
		if body.ChannelID != "C01234567" {
			t.Errorf("expected channel C01234567, got %q", body.ChannelID)
		}
		if body.Files[0].Title != "My Report" {
			t.Errorf("expected title 'My Report', got %q", body.Files[0].Title)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"files": []map[string]string{
				{"id": "F123ABC"},
			},
		})
	})

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(uploadFileParams{
		Channel:  "C01234567",
		Filename: "report.csv",
		Content:  "hello,world",
		Title:    "My Report",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.upload_file",
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
	if data["file_id"] != "F123ABC" {
		t.Errorf("expected file_id 'F123ABC', got %q", data["file_id"])
	}
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %q", data["channel"])
	}

	if !getURLCalled.Load() {
		t.Error("getUploadURLExternal was not called")
	}
	if !uploadCalled.Load() {
		t.Error("upload target was not called")
	}
	if !completeCalled.Load() {
		t.Error("completeUploadExternal was not called")
	}
}

func TestUploadFile_DefaultTitle(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	mux.HandleFunc("/files.getUploadURLExternal", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"upload_url": srv.URL + "/upload-target",
			"file_id":    "F999",
		})
	})

	mux.HandleFunc("/upload-target", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/files.completeUploadExternal", func(w http.ResponseWriter, r *http.Request) {
		var body completeUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode: %v", err)
			return
		}
		// When no title is provided, filename should be used as title.
		if len(body.Files) > 0 && body.Files[0].Title != "notes.txt" {
			t.Errorf("expected title to default to filename 'notes.txt', got %q", body.Files[0].Title)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(uploadFileParams{
		Channel:  "C01234567",
		Filename: "notes.txt",
		Content:  "some content",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.upload_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadFile_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"filename": "test.txt",
		"content":  "hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.upload_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing channel")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadFile_MissingFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
		"content": "hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.upload_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUploadFile_MissingContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel":  "C01234567",
		"filename": "test.txt",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.upload_file",
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

func TestValidateUploadURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid slack.com", "https://files.slack.com/upload/v1/abc", false},
		{"valid slack-files.com", "https://files.slack-files.com/upload/abc", false},
		{"http not https", "http://files.slack.com/upload/abc", true},
		{"arbitrary domain", "https://evil.com/upload", true},
		{"internal IP", "https://169.254.169.254/latest/meta-data/", true},
		{"localhost", "https://localhost/upload", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUploadURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUploadURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestUploadFile_GetURLError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &uploadFileAction{conn: conn}

	params, _ := json.Marshal(uploadFileParams{
		Channel:  "C01234567",
		Filename: "test.txt",
		Content:  "hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.upload_file",
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
