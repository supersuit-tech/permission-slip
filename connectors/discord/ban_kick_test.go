package discord

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestBanUser_Success(t *testing.T) {
	t.Parallel()

	conn, cleanup := mockServer(t, http.MethodPut, "/guilds/111/bans/222", http.StatusNoContent, nil)
	defer cleanup()

	action := &banUserAction{conn: conn}
	params, _ := json.Marshal(banUserParams{GuildID: "111", UserID: "222"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.ban_user",
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
	if data["status"] != "banned" {
		t.Errorf("expected status 'banned', got %q", data["status"])
	}
}

func TestBanUser_InvalidDeleteMessageSeconds(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &banUserAction{conn: conn}

	params, _ := json.Marshal(banUserParams{
		GuildID:              "111",
		UserID:               "222",
		DeleteMessageSeconds: 700000,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.ban_user",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid delete_message_seconds")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestKickUser_Success(t *testing.T) {
	t.Parallel()

	conn, cleanup := mockServer(t, http.MethodDelete, "/guilds/111/members/222", http.StatusNoContent, nil)
	defer cleanup()

	action := &kickUserAction{conn: conn}
	params, _ := json.Marshal(kickUserParams{GuildID: "111", UserID: "222"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.kick_user",
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
	if data["status"] != "kicked" {
		t.Errorf("expected status 'kicked', got %q", data["status"])
	}
}

func TestKickUser_MissingUserID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &kickUserAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"guild_id": "111"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.kick_user",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
