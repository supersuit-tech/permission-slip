package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendChannelMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/teams/team-1/channels/channel-1/messages" {
			t.Errorf("expected path /teams/team-1/channels/channel-1/messages, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		var body graphChannelMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Body.Content != "Hello Teams!" {
			t.Errorf("expected message 'Hello Teams!', got %q", body.Body.Content)
		}
		if body.Body.ContentType != "Text" {
			t.Errorf("expected content type 'Text', got %q", body.Body.ContentType)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "msg-123",
			"createdDateTime": "2024-01-15T09:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(sendChannelMessageParams{
		TeamID:    "team-1",
		ChannelID: "channel-1",
		Message:   "Hello Teams!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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
	if data["status"] != "sent" {
		t.Errorf("expected status 'sent', got %q", data["status"])
	}
	if data["message_id"] != "msg-123" {
		t.Errorf("expected message_id 'msg-123', got %q", data["message_id"])
	}
}

func TestSendChannelMessage_HTMLContent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphChannelMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Body.ContentType != "HTML" {
			t.Errorf("expected content type 'HTML' for HTML body, got %q", body.Body.ContentType)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "msg-456",
			"createdDateTime": "2024-01-15T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(sendChannelMessageParams{
		TeamID:    "team-1",
		ChannelID: "channel-1",
		Message:   "<p>Hello <strong>Teams</strong></p>",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendChannelMessage_Reply(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		expectedPath := "/teams/team-1/channels/channel-1/messages/msg-parent/replies"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "msg-reply-1",
			"createdDateTime": "2024-01-15T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(sendChannelMessageParams{
		TeamID:           "team-1",
		ChannelID:        "channel-1",
		Message:          "This is a reply",
		ReplyToMessageID: "msg-parent",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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
	if data["status"] != "sent" {
		t.Errorf("expected status 'sent', got %q", data["status"])
	}
	if data["message_id"] != "msg-reply-1" {
		t.Errorf("expected message_id 'msg-reply-1', got %q", data["message_id"])
	}
}

func TestSendChannelMessage_MissingTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel_id": "channel-1",
		"message":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing team_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChannelMessage_MissingChannelID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"team_id": "team-1",
		"message": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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

func TestSendChannelMessage_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"team_id":    "team-1",
		"channel_id": "channel-1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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

func TestSendChannelMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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

func TestSendChannelMessage_PathTraversalInTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"team_id":    "../admin",
		"channel_id": "channel-1",
		"message":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in team_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChannelMessage_SlashInChannelID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"team_id":    "team-1",
		"channel_id": "channel/../../me/sendMail",
		"message":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for slash in channel_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChannelMessage_PathTraversalInReplyID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"team_id":              "team-1",
		"channel_id":           "channel-1",
		"message":              "Hello",
		"reply_to_message_id": "../../me/sendMail",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in reply_to_message_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendChannelMessage_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token is empty.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(sendChannelMessageParams{
		TeamID:    "team-1",
		ChannelID: "channel-1",
		Message:   "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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

func TestSendChannelMessage_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChannelMessageAction{conn: conn}

	params, _ := json.Marshal(sendChannelMessageParams{
		TeamID:    "team-1",
		ChannelID: "channel-1",
		Message:   "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_channel_message",
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
