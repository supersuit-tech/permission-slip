package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListPipelines_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/pipelines/deals" {
			t.Errorf("expected path /crm/v3/pipelines/deals, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipelinesResponse{
			Results: []pipeline{
				{
					ID:    "default",
					Label: "Sales Pipeline",
					Stages: []pipelineStage{
						{ID: "appointmentscheduled", Label: "Appointment Scheduled"},
						{ID: "closedwon", Label: "Closed Won"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPipelinesAction{conn: conn}

	params, _ := json.Marshal(listPipelinesParams{})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_pipelines",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data pipelinesResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Results) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(data.Results))
	}
	if data.Results[0].ID != "default" {
		t.Errorf("expected pipeline id default, got %q", data.Results[0].ID)
	}
	if len(data.Results[0].Stages) != 2 {
		t.Errorf("expected 2 stages, got %d", len(data.Results[0].Stages))
	}
}

func TestListPipelines_TicketsObjectType(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/crm/v3/pipelines/tickets" {
			t.Errorf("expected path /crm/v3/pipelines/tickets, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pipelinesResponse{Results: []pipeline{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPipelinesAction{conn: conn}

	params, _ := json.Marshal(listPipelinesParams{ObjectType: "tickets"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_pipelines",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPipelines_InvalidObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listPipelinesAction{conn: conn}

	params, _ := json.Marshal(listPipelinesParams{ObjectType: "contacts"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_pipelines",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
