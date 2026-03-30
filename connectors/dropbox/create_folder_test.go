package dropbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateFolder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body createFolderRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Path != "/Projects/Q1" {
			t.Errorf("expected path /Projects/Q1, got %s", body.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createFolderResponse{
			Metadata: struct {
				Name        string `json:"name"`
				PathDisplay string `json:"path_display"`
				ID          string `json:"id"`
			}{
				Name:        "Q1",
				PathDisplay: "/Projects/Q1",
				ID:          "id:folder123",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createFolderAction{conn: conn}

	params, _ := json.Marshal(createFolderParams{Path: "/Projects/Q1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.create_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["name"] != "Q1" {
		t.Errorf("expected name Q1, got %s", data["name"])
	}
	if data["id"] != "id:folder123" {
		t.Errorf("expected id id:folder123, got %s", data["id"])
	}
}

func TestCreateFolder_MissingPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createFolderAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.create_folder",
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

func TestCreateFolder_RelativePath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createFolderAction{conn: conn}

	params, _ := json.Marshal(createFolderParams{Path: "no-leading-slash"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.create_folder",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for relative path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateFolder_Conflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error_summary": "path/conflict/folder",
			"error":         map[string]string{".tag": "path"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createFolderAction{conn: conn}

	params, _ := json.Marshal(createFolderParams{Path: "/existing"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.create_folder",
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
