package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateTicket_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/tickets" {
			t.Errorf("expected path /crm/v3/objects/tickets, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties["subject"] != "Login broken" {
			t.Errorf("expected subject 'Login broken', got %q", body.Properties["subject"])
		}
		if body.Properties["hs_pipeline"] != "0" {
			t.Errorf("expected hs_pipeline '0', got %q", body.Properties["hs_pipeline"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "701",
			"properties": body.Properties,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(createTicketParams{
		Subject:       "Login broken",
		Content:       "Users can't log in since the last deploy",
		Pipeline:      "0",
		PipelineStage: "1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data hubspotObjectResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "701" {
		t.Errorf("expected id 701, got %q", data.ID)
	}
}

func TestCreateTicket_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"hs_pipeline":       "0",
		"hs_pipeline_stage": "1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTicket_MissingPipeline(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject":           "Login broken",
		"hs_pipeline_stage": "1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing hs_pipeline")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTicket_MissingPipelineStage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject":     "Login broken",
		"hs_pipeline": "0",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing hs_pipeline_stage")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTicket_WithPriority(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties["hs_ticket_priority"] != "HIGH" {
			t.Errorf("expected hs_ticket_priority HIGH, got %q", body.Properties["hs_ticket_priority"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": "702", "properties": body.Properties})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(createTicketParams{
		Subject:        "Urgent bug",
		Pipeline:       "0",
		PipelineStage:  "1",
		TicketPriority: "HIGH",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
