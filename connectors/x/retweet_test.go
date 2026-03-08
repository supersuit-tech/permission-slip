package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestRetweet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/users/99/retweets" {
			t.Errorf("path = %s, want /users/99/retweets", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["tweet_id"] != "5678" {
			t.Errorf("tweet_id = %s, want 5678", body["tweet_id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"retweeted": true},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.retweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.retweet",
		Parameters:  json.RawMessage(`{"user_id":"99","tweet_id":"5678"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["retweeted"] != true {
		t.Errorf("retweeted = %v, want true", data["retweeted"])
	}
}

func TestRetweet_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.retweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.retweet",
		Parameters:  json.RawMessage(`{"user_id":"99"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUnretweet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/users/99/retweets/5678" {
			t.Errorf("path = %s, want /users/99/retweets/5678", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"retweeted": false},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.unretweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.unretweet",
		Parameters:  json.RawMessage(`{"user_id":"99","tweet_id":"5678"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["retweeted"] != false {
		t.Errorf("retweeted = %v, want false", data["retweeted"])
	}
}
