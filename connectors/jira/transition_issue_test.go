package jira

import (
	"encoding/json"
	"io"
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestTransitionIssue_WithID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/issue/PROJ-1/transitions" {
			t.Errorf("path = %s, want /issue/PROJ-1/transitions", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Transition struct {
				ID string `json:"id"`
			} `json:"transition"`
		}
		json.Unmarshal(body, &reqBody)
		if reqBody.Transition.ID != "31" {
			t.Errorf("transition.id = %q, want %q", reqBody.Transition.ID, "31")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.transition_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.transition_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","transition_id":"31"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["transition_id"] != "31" {
		t.Errorf("transition_id = %q, want %q", data["transition_id"], "31")
	}
}

func TestTransitionIssue_WithName(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: GET transitions list
			if r.Method != http.MethodGet {
				t.Errorf("call 1: method = %s, want GET", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"transitions": []map[string]string{
					{"id": "11", "name": "To Do"},
					{"id": "21", "name": "In Progress"},
					{"id": "31", "name": "Done"},
				},
			})
			return
		}
		// Second call: POST transition
		if r.Method != http.MethodPost {
			t.Errorf("call 2: method = %s, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Transition struct {
				ID string `json:"id"`
			} `json:"transition"`
		}
		json.Unmarshal(body, &reqBody)
		if reqBody.Transition.ID != "21" {
			t.Errorf("transition.id = %q, want %q (resolved from name)", reqBody.Transition.ID, "21")
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.transition_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.transition_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","transition_name":"In Progress"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["transition_id"] != "21" {
		t.Errorf("transition_id = %q, want %q", data["transition_id"], "21")
	}
}

func TestTransitionIssue_NameNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"transitions": []map[string]string{
				{"id": "11", "name": "To Do"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.transition_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.transition_issue",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","transition_name":"Nonexistent"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
	// Error should list available transitions.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "To Do") {
		t.Errorf("error message should list available transitions, got: %s", errMsg)
	}
}

func TestTransitionIssue_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.transition_issue"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing issue_key", `{"transition_id":"31"}`},
		{"missing both transition fields", `{"issue_key":"PROJ-1"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "jira.transition_issue",
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
