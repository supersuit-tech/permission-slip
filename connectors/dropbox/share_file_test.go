package dropbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestShareFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body shareRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Path != "/Documents/report.pdf" {
			t.Errorf("expected path /Documents/report.pdf, got %s", body.Path)
		}
		if body.Settings == nil || body.Settings.RequestedVisibility != "public" {
			t.Errorf("expected requested_visibility public")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shareResponse{
			URL:       "https://www.dropbox.com/s/abc123/report.pdf?dl=0",
			PathLower: "/documents/report.pdf",
			Name:      "report.pdf",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &shareFileAction{conn: conn}

	params, _ := json.Marshal(shareFileParams{
		Path:                "/Documents/report.pdf",
		RequestedVisibility: "public",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.share_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["url"] != "https://www.dropbox.com/s/abc123/report.pdf?dl=0" {
		t.Errorf("expected dropbox URL, got %s", data["url"])
	}
}

func TestShareFile_NoSettings(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body shareRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Settings != nil {
			t.Errorf("expected nil settings when no visibility specified")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shareResponse{
			URL:       "https://www.dropbox.com/s/xyz/file.txt?dl=0",
			PathLower: "/file.txt",
			Name:      "file.txt",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &shareFileAction{conn: conn}

	params, _ := json.Marshal(shareFileParams{Path: "/file.txt"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.share_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShareFile_MissingPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &shareFileAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.share_file",
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

func TestShareFile_InvalidVisibility(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &shareFileAction{conn: conn}

	params, _ := json.Marshal(shareFileParams{
		Path:                "/test.txt",
		RequestedVisibility: "invalid",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.share_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid visibility")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestShareFile_InvalidExpires(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &shareFileAction{conn: conn}

	params, _ := json.Marshal(shareFileParams{
		Path:    "/test.txt",
		Expires: "not-a-date",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.share_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid expires format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestShareFile_PasswordWithoutVisibility(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &shareFileAction{conn: conn}

	params, _ := json.Marshal(shareFileParams{
		Path:                "/test.txt",
		RequestedVisibility: "password",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.share_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for password visibility without password")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
