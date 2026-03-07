package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchTickets_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/search" {
			t.Errorf("expected path /tickets/search, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchTicketsResponse{
			Type:       "ticket.list",
			TotalCount: 1,
			Data: []intercomTicket{
				{Type: "ticket", ID: "1", Title: "Test ticket"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(searchTicketsParams{
		Field:    "state",
		Operator: "=",
		Value:    "submitted",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data searchTicketsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalCount != 1 {
		t.Errorf("expected total_count 1, got %d", data.TotalCount)
	}
}

func TestSearchTickets_MissingField(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"operator": "=",
		"value":    "test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing field")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchTickets_InvalidOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"field":    "state",
		"operator": "LIKE",
		"value":    "test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid operator")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
