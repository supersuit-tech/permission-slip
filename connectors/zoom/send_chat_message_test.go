package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendChatMessage_SuccessToJID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/users/me/messages" {
			t.Errorf("expected path /chat/users/me/messages, got %s", r.URL.Path)
		}

		var body sendChatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Message != "Hello team!" {
			t.Errorf("expected message 'Hello team!', got %q", body.Message)
		}
		if body.ToJID != "user-jid@xmpp.zoom.us" {
			t.Errorf("expected to_jid 'user-jid@xmpp.zoom.us', got %q", body.ToJID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sendChatMessageResponse{ID: "msg-abc123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(sendChatMessageParams{
		Message: "Hello team!",
		ToJID:   "user-jid@xmpp.zoom.us",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.send_chat_message",
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
	if data["id"] != "msg-abc123" {
		t.Errorf("expected id 'msg-abc123', got %q", data["id"])
	}
}

func TestSendChatMessage_SuccessToChannel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body sendChatMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.ToChannel != "channel-abc" {
			t.Errorf("expected to_channel 'channel-abc', got %q", body.ToChannel)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sendChatMessageResponse{ID: "msg-xyz789"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(sendChatMessageParams{
		Message:   "Channel announcement!",
		ToChannel: "channel-abc",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.send_chat_message",
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
	if data["id"] != "msg-xyz789" {
		t.Errorf("expected id 'msg-xyz789', got %q", data["id"])
	}
}

func TestSendChatMessage_MissingRecipient(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"message": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for missing recipient, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestSendChatMessage_BothRecipientFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"message":    "Hello",
		"to_jid":     "jid@zoom.us",
		"to_channel": "channel-abc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for both recipient fields, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestSendChatMessage_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendChatMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"to_jid": "jid@zoom.us"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.send_chat_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for missing message, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
