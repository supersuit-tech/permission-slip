package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/issue/PROJ-123" {
			t.Errorf("path = %s, want /issue/PROJ-123", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Fields map[string]interface{} `json:"fields"`
		}
		json.Unmarshal(body, &reqBody)

		if reqBody.Fields["summary"] != "Updated summary" {
			t.Errorf("summary = %v, want %q", reqBody.Fields["summary"], "Updated summary")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.update_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.update_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-123","summary":"Updated summary"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["issue_key"] != "PROJ-123" {
		t.Errorf("issue_key = %q, want %q", data["issue_key"], "PROJ-123")
	}
	if data["status"] != "updated" {
		t.Errorf("status = %q, want %q", data["status"], "updated")
	}
}

func TestUpdateIssue_MissingIssueKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.update_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.update_issue",
		Parameters:  json.RawMessage(`{"summary":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateIssue_NoFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.update_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.update_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for empty update, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateIssue_WithLabels(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Fields map[string]interface{} `json:"fields"`
		}
		json.Unmarshal(body, &reqBody)

		labels, ok := reqBody.Fields["labels"].([]interface{})
		if !ok {
			t.Fatal("expected labels array")
		}
		if len(labels) != 1 {
			t.Errorf("labels count = %d, want 1", len(labels))
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.update_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.update_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","labels":["bug"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
