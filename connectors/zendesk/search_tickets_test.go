package zendesk

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
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		query := r.URL.Query().Get("query")
		if query != "type:ticket status:open" {
			t.Errorf("expected query 'type:ticket status:open', got %q", query)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(searchResponse{
			Results: []ticket{{ID: 1, Subject: "Test"}},
			Count:   1,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(searchTicketsParams{Query: "status:open"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data searchResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 1 {
		t.Errorf("expected count 1, got %d", data.Count)
	}
}

func TestSearchTickets_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing query")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchTickets_InvalidSortBy(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchTicketsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"query":   "test",
		"sort_by": "invalid",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.search_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid sort_by")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
