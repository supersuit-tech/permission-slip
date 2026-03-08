package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMoveToSprint_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/sprint/10/issue" {
			t.Errorf("path = %s, want /sprint/10/issue", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		issues, ok := reqBody["issues"].([]interface{})
		if !ok || len(issues) != 2 {
			t.Errorf("issues = %v, want 2 items", reqBody["issues"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.move_to_sprint"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.move_to_sprint",
		Parameters:  json.RawMessage(`{"sprint_id":10,"issues":["PROJ-1","PROJ-2"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["status"] != "moved" {
		t.Errorf("status = %v, want moved", data["status"])
	}
}

func TestMoveToSprint_MissingSprintID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.move_to_sprint"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.move_to_sprint",
		Parameters:  json.RawMessage(`{"issues":["PROJ-1"]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing sprint_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestMoveToSprint_MissingIssues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.move_to_sprint"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.move_to_sprint",
		Parameters:  json.RawMessage(`{"sprint_id":10}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing issues")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
