package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchCompanies_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/organizations") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("keywords"); got != "acme" {
			t.Errorf("expected keywords='acme', got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(companySearchResponse{
			Elements: []companySearchElement{
				{ID: "org123", Name: "Acme Corp", Description: "Road runner supply", StaffCount: 500},
			},
			Paging: searchPaging{Total: 1, Start: 0, Count: 10},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &searchCompaniesAction{conn: conn}

	params, _ := json.Marshal(searchCompaniesParams{Keywords: "acme"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_companies",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	results, ok := data["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result, got: %v", data["results"])
	}
	company := results[0].(map[string]any)
	if company["id"] != "org123" {
		t.Errorf("expected id 'org123', got %v", company["id"])
	}
	if company["name"] != "Acme Corp" {
		t.Errorf("expected name 'Acme Corp', got %v", company["name"])
	}
}

func TestSearchCompanies_MissingKeywords(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchCompaniesAction{conn: conn}

	params, _ := json.Marshal(searchCompaniesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_companies",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing keywords")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchCompanies_CountTooLarge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchCompaniesAction{conn: conn}

	params, _ := json.Marshal(searchCompaniesParams{Keywords: "acme", Count: maxCompanyCount + 1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_companies",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for count too large")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchCompanies_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchCompaniesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_companies",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
