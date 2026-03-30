package jira

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
		if r.URL.Path != "/project" {
			t.Errorf("path = %s, want /project", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"key": "PROJ", "name": "My Project"},
			{"key": "ENG", "name": "Engineering"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.list_projects"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_projects",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["total_count"] != float64(2) {
		t.Errorf("total_count = %v, want 2", data["total_count"])
	}
	projects, ok := data["projects"].([]interface{})
	if !ok || len(projects) != 2 {
		t.Errorf("got %v projects, want 2", len(projects))
	}
}
