package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListContacts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/contacts/search" {
			t.Errorf("expected path /crm/v3/objects/contacts/search, got %s", r.URL.Path)
		}

		var body listContactsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Limit != 5 {
			t.Errorf("expected limit 5, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{
			Total: 1,
			Results: []hubspotObjectResponse{
				{ID: "201", Properties: map[string]string{"email": "test@example.com"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listContactsAction{conn: conn}

	params, _ := json.Marshal(listContactsParams{
		Filters: []searchFilter{{PropertyName: "email", Operator: "EQ", Value: "test@example.com"}},
		Limit:   5,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_contacts",
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
	if data.Total != 1 {
		t.Errorf("expected total 1, got %d", data.Total)
	}
	if data.Results[0].ID != "201" {
		t.Errorf("expected id 201, got %q", data.Results[0].ID)
	}
}

func TestListContacts_NoFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body listContactsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if len(body.FilterGroups) != 0 {
			t.Errorf("expected no filter groups, got %d", len(body.FilterGroups))
		}
		if len(body.Properties) != len(defaultContactProperties) {
			t.Errorf("expected %d default properties, got %d", len(defaultContactProperties), len(body.Properties))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Total: 0, Results: []hubspotObjectResponse{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listContactsAction{conn: conn}

	params, _ := json.Marshal(listContactsParams{})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_contacts",
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
	if data.Total != 0 {
		t.Errorf("expected total 0, got %d", data.Total)
	}
}

func TestListContacts_InvalidOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listContactsAction{conn: conn}

	params, _ := json.Marshal(listContactsParams{
		Filters: []searchFilter{{PropertyName: "email", Operator: "INVALID", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_contacts",
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
