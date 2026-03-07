package zendesk

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
		if r.URL.Path != "/tickets.json" {
			t.Errorf("expected path /tickets.json, got %s", r.URL.Path)
		}

		var body map[string]ticket
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if body["ticket"].Subject != "Login broken" {
			t.Errorf("expected subject 'Login broken', got %q", body["ticket"].Subject)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ticketResponse{
			Ticket: ticket{ID: 101, Subject: "Login broken"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(createTicketParams{
		Subject:     "Login broken",
		Description: "Users can't log in since the last deploy",
		Priority:    "high",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data ticketResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Ticket.ID != 101 {
		t.Errorf("expected id 101, got %d", data.Ticket.ID)
	}
}

func TestCreateTicket_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"description": "No subject provided",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_ticket",
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

func TestCreateTicket_InvalidPriority(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject":  "Test",
		"priority": "super-urgent",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTicket_InvalidType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject": "Test",
		"type":    "invalid-type",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
