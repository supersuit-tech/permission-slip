package x

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestPostTweet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/tweets" {
			t.Errorf("path = %s, want /tweets", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test_access_token_123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_access_token_123")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["text"] != "Hello world!" {
			t.Errorf("text = %q, want %q", reqBody["text"], "Hello world!")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":   "1234567890",
				"text": "Hello world!",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.post_tweet"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  json.RawMessage(`{"text":"Hello world!"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "1234567890" {
		t.Errorf("id = %v, want 1234567890", data["id"])
	}
}

func TestPostTweet_WithReply(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		reply, ok := reqBody["reply"].(map[string]any)
		if !ok {
			t.Fatal("expected reply field in request body")
		}
		if reply["in_reply_to_tweet_id"] != "9876543210" {
			t.Errorf("in_reply_to_tweet_id = %v, want 9876543210", reply["in_reply_to_tweet_id"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "111", "text": "reply"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.post_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  json.RawMessage(`{"text":"reply","reply_to_tweet_id":"9876543210"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestPostTweet_MissingText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.post_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
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

func TestPostTweet_TextTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.post_tweet"]

	longText := make([]byte, 281)
	for i := range longText {
		longText[i] = 'a'
	}

	params, _ := json.Marshal(map[string]string{"text": string(longText)})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestPostTweet_UnicodeCharacterCount(t *testing.T) {
	t.Parallel()

	// 280 emoji (each is 4 bytes = 1120 bytes total) should be accepted
	// because character count is 280, not byte count.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "1", "text": "ok"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.post_tweet"]

	// 280 emoji = 280 characters but 1120 bytes
	text := ""
	for i := 0; i < 280; i++ {
		text += "\U0001F600" // 😀
	}

	params, _ := json.Marshal(map[string]string{"text": text})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() should accept 280 emoji (280 chars), got error: %v", err)
	}
}

func TestPostTweet_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.post_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestPostTweet_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Unauthorized", "type": "about:blank"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.post_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  json.RawMessage(`{"text":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestPostTweet_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-rate-limit-reset", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"title":  "Too Many Requests",
			"detail": "Rate limit exceeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.post_tweet"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  json.RawMessage(`{"text":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestPostTweet_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.post_tweet"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "x.post_tweet",
		Parameters:  json.RawMessage(`{"text":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
