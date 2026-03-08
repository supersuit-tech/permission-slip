package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListDeals_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/deals/search" {
			t.Errorf("expected path /crm/v3/objects/deals/search, got %s", r.URL.Path)
		}

		var body listDealsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Limit != 5 {
			t.Errorf("expected limit 5, got %d", body.Limit)
		}
		if len(body.FilterGroups) != 1 || len(body.FilterGroups[0].Filters) != 1 {
			t.Fatalf("expected 1 filter group with 1 filter, got %d groups", len(body.FilterGroups))
		}
		if body.FilterGroups[0].Filters[0].PropertyName != "dealstage" {
			t.Errorf("expected filter on dealstage, got %q", body.FilterGroups[0].Filters[0].PropertyName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{
			Total: 1,
			Results: []hubspotObjectResponse{
				{ID: "101", Properties: map[string]string{"dealname": "Big Deal"}},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{
		Filters: []searchFilter{{PropertyName: "dealstage", Operator: "EQ", Value: "closedwon"}},
		Limit:   5,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
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
	if data.Results[0].ID != "101" {
		t.Errorf("expected id 101, got %q", data.Results[0].ID)
	}
}

func TestListDeals_NoFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body listDealsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if len(body.FilterGroups) != 0 {
			t.Errorf("expected no filter groups, got %d", len(body.FilterGroups))
		}
		// Should include default properties when none specified
		if len(body.Properties) != len(defaultDealProperties) {
			t.Errorf("expected %d default properties, got %d", len(defaultDealProperties), len(body.Properties))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Total: 0, Results: []hubspotObjectResponse{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
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

func TestListDeals_InvalidFilterOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{
		Filters: []searchFilter{{PropertyName: "dealstage", Operator: "INVALID", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
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

func TestListDeals_BetweenOperatorRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{
		Filters: []searchFilter{{PropertyName: "amount", Operator: "BETWEEN", Value: "100"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for BETWEEN operator (requires highValue)")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListDeals_MissingFilterValue(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{
		Filters: []searchFilter{{PropertyName: "dealstage", Operator: "EQ", Value: ""}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing filter value")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListDeals_InvalidSortDirection(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{
		Sorts: []listDealsSort{{PropertyName: "createdate", Direction: "SIDEWAYS"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid sort direction")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListDeals_LimitClamped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body listDealsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if body.Limit != maxSearchLimit {
			t.Errorf("expected limit clamped to %d, got %d", maxSearchLimit, body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Total: 0, Results: []hubspotObjectResponse{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDealsAction{conn: conn}

	params, _ := json.Marshal(listDealsParams{Limit: 9999})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.list_deals",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
