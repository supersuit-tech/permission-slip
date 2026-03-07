package intercom

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
		if r.URL.Path != "/tickets" {
			t.Errorf("expected path /tickets, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if body["title"] != "Login issue" {
			t.Errorf("expected title 'Login issue', got %q", body["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(intercomTicket{
			Type:     "ticket",
			ID:       "123",
			TicketID: "T-1001",
			Title:    "Login issue",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(createTicketParams{
		Title:        "Login issue",
		Description:  "User cannot log in",
		TicketTypeID: "type-1",
		ContactID:    "contact-abc",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_ticket",
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
	if data.ID != "123" {
		t.Errorf("expected id '123', got %q", data.ID)
	}
}

func TestCreateTicket_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ticket_type_id": "type-1",
		"contact_id":     "contact-abc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTicket_MissingTicketTypeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"title":      "Test",
		"contact_id": "contact-abc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing ticket_type_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTicket_MissingContactID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTicketAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"title":          "Test",
		"ticket_type_id": "type-1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_ticket",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing contact_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
