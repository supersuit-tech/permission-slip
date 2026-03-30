package asana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListProjects_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/projects" {
			t.Errorf("path = %s, want /projects", r.URL.Path)
		}
		if r.URL.Query().Get("workspace") != "ws123" {
			t.Errorf("workspace param = %q, want ws123", r.URL.Query().Get("workspace"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"gid": "p1", "name": "Project Alpha", "permalink_url": "https://app.asana.com/0/p1"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.list_projects"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.list_projects",
		Parameters:  json.RawMessage(`{"workspace_id":"ws123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data) != 1 {
		t.Errorf("expected 1 project, got %d", len(data))
	}
	if data[0]["name"] != "Project Alpha" {
		t.Errorf("expected name=Project Alpha, got %v", data[0]["name"])
	}
}

func TestListProjects_MissingWorkspaceID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.list_projects"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.list_projects",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
