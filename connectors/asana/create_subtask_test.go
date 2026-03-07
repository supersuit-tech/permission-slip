package asana

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSubtask_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/tasks/67890/subtasks" {
			t.Errorf("path = %s, want /tasks/67890/subtasks", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["name"] != "Sub-item" {
			t.Errorf("name = %v, want %q", data["name"], "Sub-item")
		}
		if data["assignee"] != "user@test.com" {
			t.Errorf("assignee = %v, want %q", data["assignee"], "user@test.com")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":           "11111",
				"name":          "Sub-item",
				"permalink_url": "https://app.asana.com/0/1/11111",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.create_subtask"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_subtask",
		Parameters:  json.RawMessage(`{"parent_task_id":"67890","name":"Sub-item","assignee":"user@test.com"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["gid"] != "11111" {
		t.Errorf("gid = %v, want %q", data["gid"], "11111")
	}
}

func TestCreateSubtask_MissingParentTaskID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.create_subtask"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_subtask",
		Parameters:  json.RawMessage(`{"name":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubtask_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.create_subtask"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_subtask",
		Parameters:  json.RawMessage(`{"parent_task_id":"1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
