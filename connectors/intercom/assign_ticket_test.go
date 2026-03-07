package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAssignTicket_Success(t *testing.T) {
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
		assignment, ok := body["assignment"].(map[string]any)
		if !ok {
			t.Errorf("expected assignment object in body")
			return
		}
		if assignment["admin_id"] != "admin-99" {
			t.Errorf("expected admin_id 'admin-99', got %v", assignment["admin_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(intercomTicket{
			Type: "ticket",
			ID:   "42",
			Assignee: &assignee{
				Type: "admin",
				ID:   "admin-99",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &assignTicketAction{conn: conn}

	params, _ := json.Marshal(assignTicketParams{
		TicketID:   "42",
		AssigneeID: "admin-99",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.assign_ticket",
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
	if data.Assignee == nil || data.Assignee.ID != "admin-99" {
		t.Errorf("expected assignee ID 'admin-99'")
	}
}

func TestAssignTicket_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"assignee_id": "admin-99"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.assign_ticket",
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

func TestAssignTicket_MissingAssigneeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"ticket_id": "42"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.assign_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing assignee_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAssignTicket_InvalidTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &assignTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ticket_id":   "foo/bar",
		"assignee_id": "admin-99",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.assign_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for ticket_id with path separator")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
