package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/issue" {
			t.Errorf("path = %s, want /issue", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Fields map[string]interface{} `json:"fields"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		project, _ := reqBody.Fields["project"].(map[string]interface{})
		if project["key"] != "PROJ" {
			t.Errorf("project.key = %v, want PROJ", project["key"])
		}
		if reqBody.Fields["summary"] != "Test issue" {
			t.Errorf("summary = %v, want %q", reqBody.Fields["summary"], "Test issue")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":   "10001",
			"key":  "PROJ-42",
			"self": "https://testsite.atlassian.net/rest/api/3/issue/10001",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.create_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.create_issue",
		Parameters:  json.RawMessage(`{"project_key":"PROJ","issue_type":"Bug","summary":"Test issue"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["key"] != "PROJ-42" {
		t.Errorf("key = %q, want %q", data["key"], "PROJ-42")
	}
}

func TestCreateIssue_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Fields map[string]interface{} `json:"fields"`
		}
		json.Unmarshal(body, &reqBody)

		if reqBody.Fields["description"] == nil {
			t.Error("expected description ADF, got nil")
		}
		assignee, _ := reqBody.Fields["assignee"].(map[string]interface{})
		if assignee["accountId"] != "abc123" {
			t.Errorf("assignee.accountId = %v, want abc123", assignee["accountId"])
		}
		priority, _ := reqBody.Fields["priority"].(map[string]interface{})
		if priority["name"] != "High" {
			t.Errorf("priority.name = %v, want High", priority["name"])
		}
		labels, _ := reqBody.Fields["labels"].([]interface{})
		if len(labels) != 2 {
			t.Errorf("labels count = %d, want 2", len(labels))
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "10002", "key": "PROJ-43", "self": "url"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.create_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.create_issue",
		Parameters:  json.RawMessage(`{"project_key":"PROJ","issue_type":"Story","summary":"Test","description":"Details","assignee":"abc123","priority":"High","labels":["frontend","urgent"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.create_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing project_key", `{"issue_type":"Bug","summary":"Test"}`},
		{"missing issue_type", `{"project_key":"PROJ","summary":"Test"}`},
		{"missing summary", `{"project_key":"PROJ","issue_type":"Bug"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "jira.create_issue",
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
