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
			"key": "PROJ-123",
			"fields": map[string]string{
				"summary": "Test Issue",
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
}

func TestGetIssue_WithFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fields") == "" {
			t.Error("expected fields query parameter")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"key": "PROJ-1"})
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
