package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/channels/123456789012345678/messages" {
			t.Errorf("expected path /channels/123456789012345678/messages, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bot test-bot-token-123" {
			t.Errorf("expected Bot token in Authorization header, got %q", got)
		}

		var body sendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Content != "Hello, Discord!" {
			t.Errorf("expected content 'Hello, Discord!', got %q", body.Content)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "111222333444555666",
			"channel_id": "123456789012345678",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		ChannelID: "123456789012345678",
		Content:   "Hello, Discord!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
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
	if data["id"] != "111222333444555666" {
		t.Errorf("expected id '111222333444555666', got %q", data["id"])
	}
}

func TestSendMessage_MissingChannelID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"content": "Hello"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing channel_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_MissingContent(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"channel_id": "123456789012345678"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_ContentTooLong(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &sendMessageAction{conn: conn}

	longContent := make([]byte, 2001)
	for i := range longContent {
		longContent[i] = 'a'
	}
	params, _ := json.Marshal(sendMessageParams{
		ChannelID: "123456789012345678",
		Content:   string(longContent),
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for content too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendMessage_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "401: Unauthorized",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		ChannelID: "123456789012345678",
		Content:   "Hello",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestSendMessage_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		ChannelID: "123456789012345678",
		Content:   "Hello",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestSendMessage_InvalidJSON(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &sendMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
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

func TestSendMessage_InvalidSnowflake(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &sendMessageAction{conn: conn}

	params, _ := json.Marshal(sendMessageParams{
		ChannelID: "not-a-snowflake",
		Content:   "Hello",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.send_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid snowflake")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
