package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteDriveFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/drive/v3/files/file-to-delete" {
			t.Errorf("expected path /drive/v3/files/file-to-delete, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		var body driveTrashRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if !body.Trashed {
			t.Error("expected trashed=true in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveTrashResponse{
			ID:      "file-to-delete",
			Name:    "old-report.txt",
			Trashed: true,
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{FileID: "file-to-delete"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_drive_file",
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
	if data["id"] != "file-to-delete" {
		t.Errorf("expected id 'file-to-delete', got %v", data["id"])
	}
	if data["trashed"] != true {
		t.Errorf("expected trashed=true, got %v", data["trashed"])
	}
}

func TestDeleteDriveFile_MissingFileID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_drive_file",
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

func TestDeleteDriveFile_InvalidFileID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDriveFileAction{conn: conn}

	invalidIDs := []string{
		"file/../../etc",    // path traversal
		"file id spaces",    // spaces
		"file'injection",    // quote
	}
	for _, id := range invalidIDs {
		params, _ := json.Marshal(deleteDriveFileParams{FileID: id})
		_, err := action.Execute(t.Context(), connectors.ActionRequest{
			ActionType:  "google.delete_drive_file",
			Parameters:  params,
			Credentials: validCreds(),
		})
		if err == nil {
			t.Fatalf("expected error for invalid file_id %q", id)
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("expected ValidationError for %q, got: %T", id, err)
		}
	}
}

func TestDeleteDriveFile_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 403, "message": "Insufficient permissions"},
		})
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{FileID: "file-abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_drive_file",
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

func TestDeleteDriveFile_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newDriveForTest(srv.Client(), srv.URL)
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{FileID: "file-abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_drive_file",
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

func TestDeleteDriveFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDriveFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.delete_drive_file",
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
