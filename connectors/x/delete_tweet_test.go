package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteTweet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/tweets/1234567890" {
			t.Errorf("path = %s, want /tweets/1234567890", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"deleted": true},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.delete_tweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.delete_tweet",
		Parameters:  json.RawMessage(`{"tweet_id":"1234567890"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["deleted"] != true {
		t.Errorf("deleted = %v, want true", data["deleted"])
	}
}

func TestDeleteTweet_MissingTweetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.delete_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.delete_tweet",
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

func TestDeleteTweet_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Tweet not found"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.delete_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.delete_tweet",
		Parameters:  json.RawMessage(`{"tweet_id":"9999"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
