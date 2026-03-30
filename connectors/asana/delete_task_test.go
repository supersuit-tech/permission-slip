package asana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeleteTask_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/tasks/task999" {
			t.Errorf("path = %s, want /tasks/task999", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.delete_task"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.delete_task",
		Parameters:  json.RawMessage(`{"task_id":"task999"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "deleted" {
		t.Errorf("status = %v, want deleted", data["status"])
	}
}

func TestDeleteTask_MissingTaskID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.delete_task"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.delete_task",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
