package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat.postMessage" {
			t.Errorf("expected path /chat.postMessage, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json; charset=utf-8" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		// The Slack API receives "text", not "message".
		var body sendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "#general" {
			t.Errorf("expected channel #general, got %q", body.Channel)
		}
		if body.Text != "Hello, world!" {
			t.Errorf("expected text 'Hello, world!', got %q", body.Text)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"ts":      "1234567890.123456",
			"channel": "C01234567",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		Channel: "#general",
		Message: "Hello, world!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["ts"] != "1234567890.123456" {
		t.Errorf("expected ts '1234567890.123456', got %q", data["ts"])
	}
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %q", data["channel"])
	}
}

func TestSendMessage_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"message": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing channel")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "#general",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		Channel: "#nonexistent",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for Slack API error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for channel_not_found, got: %T", err)
	}
}

func TestSendMessage_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		Channel: "#general",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for invalid_auth, got: %T", err)
	}
}

func TestSendMessage_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		Channel: "#general",
		Message: "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
	var rlErr *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rlErr) {
		if rlErr.RetryAfter.Seconds() != 30 {
			t.Errorf("expected RetryAfter 30s, got %v", rlErr.RetryAfter)
		}
	}
}

func TestSendMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.send_message",
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
