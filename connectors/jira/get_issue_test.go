package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/issue/PROJ-123" {
			t.Errorf("path = %s, want /issue/PROJ-123", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   "10001",
			"key":  "PROJ-123",
			"self": "https://example.atlassian.net/rest/api/3/issue/10001",
			"fields": map[string]interface{}{
				"summary": "Test Issue",
				"created": "2024-01-01T00:00:00.000+0000",
				"updated": "2024-01-02T00:00:00.000+0000",
				"status":  map[string]string{"name": "In Progress"},
				"assignee": map[string]string{
					"displayName": "Alice",
					"accountId":   "abc123",
				},
				"priority":  map[string]string{"name": "High"},
				"issuetype": map[string]string{"name": "Bug"},
				"labels":    []string{"backend"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.get_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.get_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["key"] != "PROJ-123" {
		t.Errorf("key = %v, want PROJ-123", data["key"])
	}
	if data["summary"] != "Test Issue" {
		t.Errorf("summary = %v, want Test Issue", data["summary"])
	}
	if data["status"] != "In Progress" {
		t.Errorf("status = %v, want In Progress", data["status"])
	}
	if data["assignee"] != "Alice" {
		t.Errorf("assignee = %v, want Alice", data["assignee"])
	}
	if data["priority"] != "High" {
		t.Errorf("priority = %v, want High", data["priority"])
	}
	if data["issue_type"] != "Bug" {
		t.Errorf("issue_type = %v, want Bug", data["issue_type"])
	}
}

func TestGetIssue_WithFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fields") == "" {
			t.Error("expected fields query parameter")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "10002",
			"key":    "PROJ-1",
			"fields": map[string]interface{}{"summary": "Test"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.get_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.get_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","fields":["summary","status"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetIssue_MissingKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.get_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.get_issue",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing issue_key")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
