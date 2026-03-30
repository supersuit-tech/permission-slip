package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetUserTweets_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/users/user123/tweets") {
			t.Errorf("path = %s, want prefix /users/user123/tweets", r.URL.Path)
		}
		if r.URL.Query().Get("max_results") != "5" {
			t.Errorf("max_results = %s, want 5", r.URL.Query().Get("max_results"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "1", "text": "tweet 1"},
				{"id": "2", "text": "tweet 2"},
			},
			"meta": map[string]any{"result_count": 2},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.get_user_tweets"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_user_tweets",
		Parameters:  json.RawMessage(`{"user_id":"user123","max_results":5}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	tweets, ok := data["data"].([]any)
	if !ok {
		t.Fatal("expected data array in response")
	}
	if len(tweets) != 2 {
		t.Errorf("got %d tweets, want 2", len(tweets))
	}
}

func TestGetUserTweets_DefaultMaxResults(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("max_results") != "10" {
			t.Errorf("max_results = %s, want 10 (default)", r.URL.Query().Get("max_results"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.get_user_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_user_tweets",
		Parameters:  json.RawMessage(`{"user_id":"user123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetUserTweets_MissingUserID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.get_user_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_user_tweets",
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

func TestGetUserTweets_WithPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("since_id") != "100" {
			t.Errorf("since_id = %s, want 100", r.URL.Query().Get("since_id"))
		}
		if r.URL.Query().Get("until_id") != "200" {
			t.Errorf("until_id = %s, want 200", r.URL.Query().Get("until_id"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.get_user_tweets"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.get_user_tweets",
		Parameters:  json.RawMessage(`{"user_id":"user123","since_id":"100","until_id":"200"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
