package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestTriggerWorkflow_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/repos/octocat/hello-world/actions/workflows/deploy.yml/dispatches" {
			t.Errorf("path = %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if reqBody["ref"] != "main" {
			t.Errorf("ref = %v, want main", reqBody["ref"])
		}

		// GitHub returns 204 No Content on success
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.trigger_workflow"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.trigger_workflow",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","workflow_id":"deploy.yml","ref":"main"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "dispatched" {
		t.Errorf("status = %v, want dispatched", data["status"])
	}
}

func TestTriggerWorkflow_WithInputs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		inputs, ok := reqBody["inputs"].(map[string]any)
		if !ok {
			t.Fatal("inputs not present or not a map")
		}
		if inputs["environment"] != "production" {
			t.Errorf("inputs.environment = %v", inputs["environment"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.trigger_workflow"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.trigger_workflow",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","workflow_id":"deploy.yml","ref":"main","inputs":{"environment":"production"}}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestTriggerWorkflow_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.trigger_workflow"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","workflow_id":"deploy.yml","ref":"main"}`},
		{"missing repo", `{"owner":"o","workflow_id":"deploy.yml","ref":"main"}`},
		{"missing workflow_id", `{"owner":"o","repo":"r","ref":"main"}`},
		{"missing ref", `{"owner":"o","repo":"r","workflow_id":"deploy.yml"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.trigger_workflow",
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
