package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchTweets_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Query().Get("query") != "golang" {
			t.Errorf("query = %s, want golang", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("max_results") != "20" {
			t.Errorf("max_results = %s, want 20", r.URL.Query().Get("max_results"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "1", "text": "Go is great"},
			},
			"meta": map[string]any{"result_count": 1},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.search_tweets"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.search_tweets",
		Parameters:  json.RawMessage(`{"query":"golang","max_results":20}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result.Data == nil {
		t.Fatal("expected non-nil result data")
	}
}

func TestSearchTweets_WithSortOrder(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sort_order") != "recency" {
			t.Errorf("sort_order = %s, want recency", r.URL.Query().Get("sort_order"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.search_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.search_tweets",
		Parameters:  json.RawMessage(`{"query":"test","sort_order":"recency"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchTweets_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.search_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.search_tweets",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchTweets_MaxResultsTooLow(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.search_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.search_tweets",
		Parameters:  json.RawMessage(`{"query":"test","max_results":5}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for max_results=5, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchTweets_InvalidSortOrder(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.search_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.search_tweets",
		Parameters:  json.RawMessage(`{"query":"test","sort_order":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
