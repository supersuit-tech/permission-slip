package asana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListSections_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/projects/proj1/sections" {
			t.Errorf("path = %s, want /projects/proj1/sections", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"gid": "s1", "name": "To Do"},
				{"gid": "s2", "name": "In Progress"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.list_sections"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.list_sections",
		Parameters:  json.RawMessage(`{"project_id":"proj1"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 sections, got %d", len(data))
	}
}

func TestListSections_MissingProjectID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.list_sections"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.list_sections",
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
