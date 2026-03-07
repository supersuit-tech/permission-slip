package notion

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearch_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/search" {
			t.Errorf("expected path /v1/search, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["query"] != "meeting notes" {
			t.Errorf("expected query 'meeting notes', got %v", body["query"])
		}
		if body["page_size"] != float64(100) {
			t.Errorf("expected page_size 100, got %v", body["page_size"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"results":  []map[string]any{{"object": "page", "id": "page-1"}},
			"has_more": false,
		})
	})

	action := &searchAction{conn: conn}
	params, _ := json.Marshal(searchParams{
		Query: "meeting notes",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.search",
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
	if !ok {
		t.Fatal("expected results array")
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestSearch_WithFilter(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		filter, ok := body["filter"].(map[string]any)
		if !ok {
			t.Fatal("expected filter in request body")
		}
		if filter["value"] != "page" {
			t.Errorf("expected filter value 'page', got %v", filter["value"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"object": "list", "results": []any{}, "has_more": false})
	})

	action := &searchAction{conn: conn}
	filter := `{"property":"object","value":"page"}`
	params, _ := json.Marshal(searchParams{
		Query:  "project",
		Filter: json.RawMessage(filter),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.search",
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
	params, _ := json.Marshal(map[string]any{"page_size": 10})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.search",
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

func TestSearch_InvalidPageSize(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"query": "test", "page_size": 200})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.search",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid page_size")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearch_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.search",
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
