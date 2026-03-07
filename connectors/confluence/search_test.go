package confluence

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
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("cql"); got != "type=page AND space=DEV" {
			t.Errorf("cql = %q, want %q", got, "type=page AND space=DEV")
		}
		if got := r.URL.Query().Get("limit"); got != "25" {
			t.Errorf("limit = %q, want 25", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"content": map[string]interface{}{"id": "1", "title": "Page 1"}, "excerpt": "..."},
				{"content": map[string]interface{}{"id": "2", "title": "Page 2"}, "excerpt": "..."},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.search"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.search",
		Parameters:  json.RawMessage(`{"cql":"type=page AND space=DEV"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	results, ok := data["results"].([]interface{})
	if !ok {
		t.Fatal("expected results array")
	}
	if len(results) != 2 {
		t.Errorf("results count = %d, want 2", len(results))
	}
}

func TestSearch_CustomLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want 10", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.search",
		Parameters:  json.RawMessage(`{"cql":"type=page","limit":10}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearch_LimitExceeded(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["confluence.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.search",
		Parameters:  json.RawMessage(`{"cql":"type=page","limit":500}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for excessive limit, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearch_MissingCQL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["confluence.search"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.search",
		Parameters:  json.RawMessage(`{"limit":10}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
