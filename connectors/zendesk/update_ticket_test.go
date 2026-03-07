package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateTicket_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/42.json" {
			t.Errorf("expected path /tickets/42.json, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		tkt, ok := body["ticket"].(map[string]any)
		if !ok {
			t.Fatal("expected ticket object in body")
		}
		if tkt["status"] != "solved" {
			t.Errorf("expected status 'solved', got %v", tkt["status"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticketResponse{
			Ticket: ticket{ID: 42, Status: "solved"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(updateTicketParams{
		TicketID: 42,
		Status:   "solved",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_ticket",
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
	if data.Ticket.Status != "solved" {
		t.Errorf("expected status 'solved', got %q", data.Ticket.Status)
	}
}

func TestUpdateTicket_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"status": "open"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing ticket_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateTicket_NoFieldsProvided(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]int64{"ticket_id": 42})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no update fields provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateTicket_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(updateTicketParams{
		TicketID: 42,
		Status:   "invalid_status",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
