package figma

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetFile_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/abc123" {
			t.Errorf("expected path /files/abc123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":     "My Design",
			"document": map[string]any{"id": "0:0", "type": "DOCUMENT"},
		})
	})

	action := &getFileAction{conn: conn}
	params, _ := json.Marshal(getFileParams{FileKey: "abc123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_file",
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
	if data["name"] != "My Design" {
		t.Errorf("expected name 'My Design', got %v", data["name"])
	}
}

func TestGetFile_WithDepthAndNodeIDs(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("depth"); got != "2" {
			t.Errorf("expected depth=2, got %q", got)
		}
		if got := r.URL.Query().Get("ids"); got != "1:2,3:4" {
			t.Errorf("expected ids=1:2,3:4, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"name": "Test"})
	})

	action := &getFileAction{conn: conn}
	params, _ := json.Marshal(getFileParams{
		FileKey: "abc123",
		Depth:   2,
		NodeIDs: "1:2,3:4",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFile_MissingFileKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getFileAction{conn: conn}
	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing file_key")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetFile_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getFileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_file",
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

func TestGetFile_NegativeDepth(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getFileAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"file_key": "abc123", "depth": -1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_file",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative depth")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
