package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteDriveFile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/me/drive/items/file-123" {
			t.Errorf("expected path /me/drive/items/file-123, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{ItemID: "file-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.delete_drive_file",
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
	if data["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %q", data["status"])
	}
	if data["item_id"] != "file-123" {
		t.Errorf("expected item_id 'file-123', got %q", data["item_id"])
	}
	if data["message"] == "" {
		t.Error("expected non-empty message")
	}
}

func TestDeleteDriveFile_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.delete_drive_file",
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

func TestDeleteDriveFile_InvalidItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDriveFileAction{conn: conn}

	cases := []struct {
		name   string
		itemID string
	}{
		{"path-traversal", "../../../etc"},
		{"path-separator-slash", "a/b"},
		{"path-separator-backslash", "a\\b"},
		{"query-injection", "file-123?$expand=malicious"},
		{"fragment-injection", "file-123#fragment"},
		{"percent-encoding", "file-123%2F..%2Fetc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params, _ := json.Marshal(deleteDriveFileParams{ItemID: tc.itemID})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "microsoft.delete_drive_file",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error for invalid item_id")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestDeleteDriveFile_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token is empty.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{ItemID: "file-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.delete_drive_file",
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

func TestDeleteDriveFile_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "itemNotFound",
				"message": "The resource could not be found.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteDriveFileAction{conn: conn}

	params, _ := json.Marshal(deleteDriveFileParams{ItemID: "nonexistent"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.delete_drive_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestDeleteDriveFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDriveFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.delete_drive_file",
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
