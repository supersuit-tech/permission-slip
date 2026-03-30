package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchContacts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/contacts/search" {
			t.Errorf("expected path /contacts/search, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(contactsSearchResponse{
			Type:       "list",
			TotalCount: 1,
			Data: []intercomContact{
				{Type: "contact", ID: "abc123", Email: "user@example.com"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchContactsAction{conn: conn}

	params, _ := json.Marshal(searchContactsParams{
		Query: intercomSearchQuery{
			Field:    "email",
			Operator: "=",
			Value:    "user@example.com",
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_contacts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data contactsSearchResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalCount != 1 {
		t.Errorf("expected total_count 1, got %d", data.TotalCount)
	}
}

func TestSearchContacts_MissingField(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchContactsAction{conn: conn}

	params, _ := json.Marshal(searchContactsParams{
		Query: intercomSearchQuery{Operator: "=", Value: "x"},
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_contacts",
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

func TestSearchContacts_InvalidOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchContactsAction{conn: conn}

	params, _ := json.Marshal(searchContactsParams{
		Query: intercomSearchQuery{Field: "email", Operator: "INVALID", Value: "x"},
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.search_contacts",
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
