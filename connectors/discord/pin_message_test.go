package discord

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestPinMessage_Success(t *testing.T) {
	t.Parallel()

	conn, cleanup := mockServer(t, http.MethodPut, "/channels/111/pins/222", http.StatusNoContent, nil)
	defer cleanup()

	action := &pinMessageAction{conn: conn}
	params, _ := json.Marshal(pinMessageParams{ChannelID: "111", MessageID: "222"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.pin_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if data["status"] != "pinned" {
		t.Errorf("expected status 'pinned', got %q", data["status"])
	}
}

func TestUnpinMessage_Success(t *testing.T) {
	t.Parallel()

	conn, cleanup := mockServer(t, http.MethodDelete, "/channels/111/pins/222", http.StatusNoContent, nil)
	defer cleanup()

	action := &unpinMessageAction{conn: conn}
	params, _ := json.Marshal(pinMessageParams{ChannelID: "111", MessageID: "222"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.unpin_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if data["status"] != "unpinned" {
		t.Errorf("expected status 'unpinned', got %q", data["status"])
	}
}

func TestPinMessage_MissingMessageID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &pinMessageAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"channel_id": "111"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.pin_message",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
