package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchPeople_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/people") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("q"); got != "search" {
			t.Errorf("expected q=search, got %q", got)
		}
		if got := r.URL.Query().Get("keywords"); got != "software engineer" {
			t.Errorf("expected keywords='software engineer', got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(peopleSearchResponse{
			Elements: []peopleSearchElement{
				{ID: "abc123", FirstName: "Jane", LastName: "Doe", Headline: "Software Engineer at Acme"},
			},
			Paging: searchPaging{Total: 1, Start: 0, Count: 10},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &searchPeopleAction{conn: conn}

	params, _ := json.Marshal(searchPeopleParams{Keywords: "software engineer"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_people",
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
	person := results[0].(map[string]any)
	if person["id"] != "abc123" {
		t.Errorf("expected id 'abc123', got %v", person["id"])
	}
	if person["first_name"] != "Jane" {
		t.Errorf("expected first_name 'Jane', got %v", person["first_name"])
	}
	if person["person_urn"] != "urn:li:person:abc123" {
		t.Errorf("expected person_urn 'urn:li:person:abc123', got %v", person["person_urn"])
	}
	// next_start should equal start (0) + len(results) (1) = 1
	if ns, ok := data["next_start"].(float64); !ok || ns != 1 {
		t.Errorf("expected next_start 1, got %v", data["next_start"])
	}
}

func TestSearchPeople_NoSearchTerms(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchPeopleAction{conn: conn}

	params, _ := json.Marshal(searchPeopleParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_people",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no search terms provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchPeople_CountTooLarge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchPeopleAction{conn: conn}

	params, _ := json.Marshal(searchPeopleParams{Keywords: "engineer", Count: maxSearchCount + 1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_people",
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

func TestSearchPeople_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchPeopleAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.search_people",
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
