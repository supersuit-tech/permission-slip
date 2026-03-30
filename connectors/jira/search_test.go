package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearch_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Errorf("path = %s, want /search", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		if reqBody["jql"] != "project = PROJ" {
			t.Errorf("jql = %v, want %q", reqBody["jql"], "project = PROJ")
		}
		if reqBody["maxResults"] != float64(50) {
			t.Errorf("maxResults = %v, want 50", reqBody["maxResults"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total":      2,
			"maxResults": 50,
			"issues": []map[string]interface{}{
				{"key": "PROJ-1", "fields": map[string]string{"summary": "Issue 1"}},
				{"key": "PROJ-2", "fields": map[string]string{"summary": "Issue 2"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.search"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.search",
		Parameters:  json.RawMessage(`{"jql":"project = PROJ"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["total"] != float64(2) {
		t.Errorf("total = %v, want 2", data["total"])
	}
}

func TestSearch_CustomMaxResults(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		if reqBody["maxResults"] != float64(10) {
			t.Errorf("maxResults = %v, want 10", reqBody["maxResults"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"total": 0, "issues": []interface{}{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.search",
		Parameters:  json.RawMessage(`{"jql":"project = PROJ","max_results":10}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearch_WithFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		fields, ok := reqBody["fields"].([]interface{})
		if !ok {
			t.Fatal("expected fields array")
		}
		if len(fields) != 2 {
			t.Errorf("fields count = %d, want 2", len(fields))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"total": 0, "issues": []interface{}{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.search",
		Parameters:  json.RawMessage(`{"jql":"assignee = me","fields":["summary","status"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearch_MaxResultsExceeded(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.search",
		Parameters:  json.RawMessage(`{"jql":"project = PROJ","max_results":5000}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for excessive max_results, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearch_MissingJQL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.search",
		Parameters:  json.RawMessage(`{"max_results":10}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
