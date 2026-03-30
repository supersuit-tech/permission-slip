package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListStatuses_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/status" {
			t.Errorf("path = %s, want /status", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"id": "1", "name": "To Do"},
			{"id": "2", "name": "In Progress"},
			{"id": "3", "name": "Done"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.list_statuses"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_statuses",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(data) != 3 {
		t.Errorf("got %d statuses, want 3", len(data))
	}
}

func TestListStatuses_WithProjectKey(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("projectKey") != "PROJ" {
			t.Errorf("projectKey = %q, want PROJ", r.URL.Query().Get("projectKey"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.list_statuses"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_statuses",
		Parameters:  json.RawMessage(`{"project_key":"PROJ"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
