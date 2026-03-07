package intercom

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
		if r.URL.Path != "/tickets/42" {
			t.Errorf("expected path /tickets/42, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if body["state"] != "resolved" {
			t.Errorf("expected state 'resolved', got %v", body["state"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(intercomTicket{
			Type:  "ticket",
			ID:    "42",
			State: "resolved",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(updateTicketParams{
		TicketID: "42",
		State:    "resolved",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data intercomTicket
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.State != "resolved" {
		t.Errorf("expected state 'resolved', got %q", data.State)
	}
}

func TestUpdateTicket_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"state": "resolved"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_ticket",
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

	params, _ := json.Marshal(map[string]string{"ticket_id": "42"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_ticket",
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

func TestUpdateTicket_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ticket_id": "42",
		"state":     "invalid_state",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateTicket_InvalidTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ticket_id": "../../admin",
		"state":     "resolved",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid ticket_id with path traversal")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
