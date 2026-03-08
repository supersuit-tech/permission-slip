package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListWorkflowRuns_AllWorkflows(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/actions/runs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"total_count": 1,
			"workflow_runs": []map[string]any{
				{
					"id":          100,
					"name":        "CI",
					"status":      "completed",
					"conclusion":  "success",
					"html_url":    "https://github.com/octocat/hello-world/actions/runs/100",
					"head_branch": "main",
					"head_sha":    "abc123",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_workflow_runs"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_workflow_runs",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["total_count"] != float64(1) {
		t.Errorf("total_count = %v, want 1", data["total_count"])
	}
}

func TestListWorkflowRuns_SpecificWorkflow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/actions/workflows/ci.yml/runs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("status") != "success" {
			t.Errorf("status = %q, want success", r.URL.Query().Get("status"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"total_count":   0,
			"workflow_runs": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.list_workflow_runs"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.list_workflow_runs",
		Parameters:  json.RawMessage(`{"owner":"octocat","repo":"hello-world","workflow_id":"ci.yml","status":"success"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListWorkflowRuns_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.list_workflow_runs"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"hello-world"}`},
		{"missing repo", `{"owner":"octocat"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.list_workflow_runs",
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
