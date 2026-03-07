package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReplyTicket_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/42.json" {
			t.Errorf("expected path /tickets/42.json, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ticketResponse{
			Ticket: ticket{ID: 42, Subject: "Original ticket"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &replyTicketAction{conn: conn}

	params, _ := json.Marshal(replyTicketParams{
		TicketID: 42,
		Body:     "Thanks for reaching out! We're looking into this.",
		Public:   true,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.reply_ticket",
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
	if data.Ticket.ID != 42 {
		t.Errorf("expected id 42, got %d", data.Ticket.ID)
	}
}

func TestReplyTicket_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"body": "hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.reply_ticket",
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

func TestReplyTicket_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"ticket_id": 42})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.reply_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
