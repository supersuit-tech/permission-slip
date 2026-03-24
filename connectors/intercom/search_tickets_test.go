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

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		q, ok := body["query"].(map[string]any)
		if !ok || q["field"] != "state" {
			t.Errorf("expected simple query with field state, got %#v", body["query"])
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

func TestSearchTickets_WithCreatedAtBounds_ANDQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		top, ok := body["query"].(map[string]any)
		if !ok || top["operator"] != "AND" {
			t.Fatalf("expected AND query, got %#v", body["query"])
		}
		vals, ok := top["value"].([]any)
		if !ok || len(vals) != 2 {
			t.Fatalf("expected 2 predicates, got %#v", top["value"])
		}
		p0 := vals[0].(map[string]any)
		if p0["field"] != "state" || p0["operator"] != "=" {
			t.Errorf("unexpected first predicate: %#v", p0)
		}
		p1 := vals[1].(map[string]any)
		if p1["field"] != "created_at" || p1["operator"] != ">" || p1["value"] != "1709251200" {
			t.Errorf("unexpected second predicate: %#v", p1)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchTicketsResponse{Type: "ticket.list", TotalCount: 0, Data: []intercomTicket{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(searchTicketsParams{
		Field:          "state",
		Operator:       "=",
		Value:          "submitted",
		CreatedAtAfter: "1709251200",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
