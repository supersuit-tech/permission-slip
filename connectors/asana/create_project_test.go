package asana

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateProject_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/projects" {
			t.Errorf("path = %s, want /projects", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["name"] != "New Project" {
			t.Errorf("name = %v, want New Project", data["name"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":           "proj123",
				"name":          "New Project",
				"permalink_url": "https://app.asana.com/0/proj123",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.create_project"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_project",
		Parameters:  json.RawMessage(`{"workspace_id":"ws1","name":"New Project"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["gid"] != "proj123" {
		t.Errorf("gid = %v, want proj123", data["gid"])
	}
}

func TestCreateProject_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.create_project"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_project",
		Parameters:  json.RawMessage(`{"workspace_id":"ws1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
