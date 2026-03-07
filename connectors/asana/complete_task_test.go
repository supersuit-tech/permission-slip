package asana

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCompleteTask_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/tasks/67890" {
			t.Errorf("path = %s, want /tasks/67890", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["completed"] != true {
			t.Errorf("completed = %v, want true", data["completed"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":           "67890",
				"name":          "Some task",
				"completed":     true,
				"permalink_url": "https://app.asana.com/0/1/67890",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.complete_task"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.complete_task",
		Parameters:  json.RawMessage(`{"task_id":"67890"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["completed"] != true {
		t.Errorf("completed = %v, want true", data["completed"])
	}
	if data["gid"] != "67890" {
		t.Errorf("gid = %v, want %q", data["gid"], "67890")
	}
}

func TestCompleteTask_MissingTaskID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.complete_task"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.complete_task",
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
