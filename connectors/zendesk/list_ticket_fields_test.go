package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListTicketFields_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/ticket_fields.json" {
			t.Errorf("expected path /ticket_fields.json, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ticketFieldsResponse{
			Count: 2,
			TicketFields: []ticketField{
				{ID: 1, Type: "subject", Title: "Subject", Active: true},
				{ID: 2, Type: "text", Title: "Order ID", Active: true},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTicketFieldsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.list_ticket_fields",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data ticketFieldsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 2 {
		t.Errorf("expected count 2, got %d", data.Count)
	}
	if len(data.TicketFields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(data.TicketFields))
	}
}
