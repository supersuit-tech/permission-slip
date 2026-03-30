package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAssignTicket_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/42.json" {
			t.Errorf("expected path /tickets/42.json, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticketResponse{
			Ticket: ticket{ID: 42, AssigneeID: int64Ptr(99)},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &assignTicketAction{conn: conn}

	assignee := int64(99)
	params, _ := json.Marshal(assignTicketParams{
		TicketID:   42,
		AssigneeID: &assignee,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.assign_ticket",
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
	if data.Ticket.AssigneeID == nil || *data.Ticket.AssigneeID != 99 {
		t.Error("expected assignee_id 99")
	}
}

func TestAssignTicket_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignTicketAction{conn: conn}

	assignee := int64(99)
	params, _ := json.Marshal(assignTicketParams{AssigneeID: &assignee})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.assign_ticket",
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

func TestAssignTicket_NeitherAssigneeNorGroup(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]int64{"ticket_id": 42})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.assign_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when neither assignee_id nor group_id provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAssignTicket_InvalidAssigneeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignTicketAction{conn: conn}

	badID := int64(-1)
	params, _ := json.Marshal(assignTicketParams{
		TicketID:   42,
		AssigneeID: &badID,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.assign_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative assignee_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func int64Ptr(v int64) *int64 { return &v }
