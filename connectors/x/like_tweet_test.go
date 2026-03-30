package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestLikeTweet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/users/99/likes" {
			t.Errorf("path = %s, want /users/99/likes", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["tweet_id"] != "1234" {
			t.Errorf("tweet_id = %s, want 1234", body["tweet_id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"liked": true},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.like_tweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.like_tweet",
		Parameters:  json.RawMessage(`{"user_id":"99","tweet_id":"1234"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["liked"] != true {
		t.Errorf("liked = %v, want true", data["liked"])
	}
}

// TestLikeTweet_AutoResolveUserID verifies that omitting user_id triggers
// a /users/me lookup and uses the returned ID for the likes endpoint.
func TestLikeTweet_AutoResolveUserID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/me" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": "42", "name": "Test User", "username": "testuser"},
			})
			return
		}
		if r.URL.Path == "/users/42/likes" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"liked": true},
			})
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.like_tweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.like_tweet",
		Parameters:  json.RawMessage(`{"tweet_id":"1234"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["liked"] != true {
		t.Errorf("liked = %v, want true", data["liked"])
	}
}

func TestLikeTweet_MissingTweetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.like_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.like_tweet",
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

func TestUnlikeTweet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/users/99/likes/1234" {
			t.Errorf("path = %s, want /users/99/likes/1234", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"liked": false},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.unlike_tweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.unlike_tweet",
		Parameters:  json.RawMessage(`{"user_id":"99","tweet_id":"1234"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["liked"] != false {
		t.Errorf("liked = %v, want false", data["liked"])
	}
}
