package dropbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMove_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body moveRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.FromPath != "/old/file.txt" {
			t.Errorf("expected from_path /old/file.txt, got %s", body.FromPath)
		}
		if body.ToPath != "/new/file.txt" {
			t.Errorf("expected to_path /new/file.txt, got %s", body.ToPath)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(moveResponse{
			Metadata: struct {
				Name        string `json:"name"`
				PathDisplay string `json:"path_display"`
				ID          string `json:"id"`
			}{
				Name:        "file.txt",
				PathDisplay: "/new/file.txt",
				ID:          "id:moved123",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &moveAction{conn: conn}

	params, _ := json.Marshal(moveParams{
		FromPath: "/old/file.txt",
		ToPath:   "/new/file.txt",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.move",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["path_display"] != "/new/file.txt" {
		t.Errorf("expected path_display /new/file.txt, got %s", data["path_display"])
	}
}

func TestMove_MissingFromPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &moveAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"to_path": "/new/file.txt"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.move",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing from_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMove_MissingToPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &moveAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"from_path": "/old/file.txt"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.move",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing to_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMove_RelativePaths(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &moveAction{conn: conn}

	params, _ := json.Marshal(moveParams{
		FromPath: "relative/path",
		ToPath:   "/valid/path",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.move",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for relative from_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMove_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"error_summary": "cant_move_folder_into_itself/...",
			"error":         map[string]string{".tag": "cant_move_folder_into_itself"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &moveAction{conn: conn}

	params, _ := json.Marshal(moveParams{
		FromPath: "/folder",
		ToPath:   "/folder/subfolder",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.move",
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
