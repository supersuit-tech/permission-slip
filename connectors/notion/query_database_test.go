package notion

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestQueryDatabase_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/databases/db-123/query" {
			t.Errorf("expected path /v1/databases/db-123/query, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["page_size"] != float64(100) {
			t.Errorf("expected page_size 100, got %v", body["page_size"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"object":   "list",
			"results":  []map[string]any{{"object": "page", "id": "page-1"}, {"object": "page", "id": "page-2"}},
			"has_more": false,
		})
	})

	action := &queryDatabaseAction{conn: conn}
	params, _ := json.Marshal(queryDatabaseParams{
		DatabaseID: "db-123",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.query_database",
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
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestQueryDatabase_WithFilter(t *testing.T) {
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
		if filter["property"] != "Status" {
			t.Errorf("expected filter property 'Status', got %v", filter["property"])
		}
		if body["page_size"] != float64(50) {
			t.Errorf("expected page_size 50, got %v", body["page_size"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"object": "list", "results": []any{}, "has_more": false})
	})

	action := &queryDatabaseAction{conn: conn}
	filter := `{"property":"Status","select":{"equals":"Done"}}`
	params, _ := json.Marshal(queryDatabaseParams{
		DatabaseID: "db-123",
		Filter:     json.RawMessage(filter),
		PageSize:   50,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.query_database",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueryDatabase_MissingDatabaseID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &queryDatabaseAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"page_size": 10})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.query_database",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing database_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestQueryDatabase_InvalidPageSize(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &queryDatabaseAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"database_id": "db-123", "page_size": 200})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.query_database",
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

func TestQueryDatabase_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &queryDatabaseAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.query_database",
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
