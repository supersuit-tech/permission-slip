package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAssignIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/issue/PROJ-1/assignee" {
			t.Errorf("path = %s, want /issue/PROJ-1/assignee", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		json.Unmarshal(body, &reqBody)

		if reqBody["accountId"] != "abc123" {
			t.Errorf("accountId = %q, want %q", reqBody["accountId"], "abc123")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.assign_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.assign_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","account_id":"abc123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["status"] != "assigned" {
		t.Errorf("status = %q, want %q", data["status"], "assigned")
	}
	if data["account_id"] != "abc123" {
		t.Errorf("account_id = %q, want %q", data["account_id"], "abc123")
	}
}

func TestAssignIssue_Unassign(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		// Jira expects {"accountId": null} to unassign.
		if _, exists := reqBody["accountId"]; !exists {
			t.Error("expected accountId key in request body")
		}
		if reqBody["accountId"] != nil {
			t.Errorf("accountId = %v, want nil", reqBody["accountId"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.assign_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.assign_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["status"] != "unassigned" {
		t.Errorf("status = %q, want %q", data["status"], "unassigned")
	}
}

func TestAssignIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.assign_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing issue_key", `{"account_id":"abc123"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "jira.assign_issue",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
