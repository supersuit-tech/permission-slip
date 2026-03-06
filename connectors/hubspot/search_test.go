package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearch_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/contacts/search" {
			t.Errorf("expected path /crm/v3/objects/contacts/search, got %s", r.URL.Path)
		}

		var body searchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.FilterGroups) != 1 {
			t.Fatalf("expected 1 filter group, got %d", len(body.FilterGroups))
		}
		if len(body.FilterGroups[0].Filters) != 1 {
			t.Fatalf("expected 1 filter, got %d", len(body.FilterGroups[0].Filters))
		}
		f := body.FilterGroups[0].Filters[0]
		if f.PropertyName != "email" || f.Operator != "EQ" || f.Value != "jane@example.com" {
			t.Errorf("unexpected filter: %+v", f)
		}
		if body.Limit != 10 {
			t.Errorf("expected limit 10, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"results": []map[string]any{
				{
					"id":         "501",
					"properties": map[string]string{"email": "jane@example.com"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "contacts",
		Filters: []searchFilter{
			{PropertyName: "email", Operator: "EQ", Value: "jane@example.com"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
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
	if len(data.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(data.Results))
	}
	if data.Results[0].ID != "501" {
		t.Errorf("expected id 501, got %q", data.Results[0].ID)
	}
}

func TestSearch_CustomLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Limit != 5 {
			t.Errorf("expected limit 5, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"total": 0, "results": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "deals",
		Filters:    []searchFilter{{PropertyName: "dealname", Operator: "CONTAINS_TOKEN", Value: "big"}},
		Limit:      5,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_MissingObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"filters": []searchFilter{{PropertyName: "email", Operator: "EQ", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_InvalidObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "widgets",
		Filters:    []searchFilter{{PropertyName: "name", Operator: "EQ", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_EmptyFilters(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "contacts",
		Filters:    []searchFilter{},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for empty filters")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_FilterMissingPropertyName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "contacts",
		Filters:    []searchFilter{{Operator: "EQ", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing propertyName")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_InvalidOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "contacts",
		Filters:    []searchFilter{{PropertyName: "email", Operator: "LIKE", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
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

func TestSearch_LimitCappedAt200(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Limit != 200 {
			t.Errorf("expected limit to be capped at 200, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"total": 0, "results": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "contacts",
		Filters:    []searchFilter{{PropertyName: "email", Operator: "EQ", Value: "x"}},
		Limit:      500, // exceeds HubSpot's max of 200
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "error",
			"category": "UNAUTHORIZED",
			"message":  "Invalid access token",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		ObjectType: "contacts",
		Filters:    []searchFilter{{PropertyName: "email", Operator: "EQ", Value: "x"}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
