package dropbox

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

		var body searchRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Query != "budget" {
			t.Errorf("expected query 'budget', got %s", body.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{
			Matches: []searchMatch{
				{
					Metadata: searchMatchMetadata{
						Metadata: searchFileMetadata{
							Tag:         "file",
							Name:        "budget.xlsx",
							PathDisplay: "/Finance/budget.xlsx",
							ID:          "id:file123",
							Size:        2048,
						},
					},
				},
			},
			HasMore: false,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{Query: "budget"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	matches := data["matches"].([]any)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	match := matches[0].(map[string]any)
	if match["name"] != "budget.xlsx" {
		t.Errorf("expected name budget.xlsx, got %v", match["name"])
	}
}

func TestSearch_WithOptions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body searchRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Options == nil {
			t.Fatal("expected options to be set")
		}
		if body.Options.Path != "/Documents" {
			t.Errorf("expected path /Documents, got %s", body.Options.Path)
		}
		if body.Options.MaxResults != 5 {
			t.Errorf("expected max_results 5, got %d", body.Options.MaxResults)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Matches: []searchMatch{}, HasMore: false})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		Query:      "report",
		Path:       "/Documents",
		MaxResults: 5,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_MissingQuery(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.search",
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

func TestSearch_QueryTooShort(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{Query: "ab"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for short query")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_InvalidPath(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		Query: "test query",
		Path:  "no-leading-slash",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_InvalidMaxResults(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &searchAction{conn: conn}

	params, _ := json.Marshal(searchParams{
		Query:      "test",
		MaxResults: 5000,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "dropbox.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid max_results")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
