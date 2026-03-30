package discord

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListChannels_Success(t *testing.T) {
	t.Parallel()

	conn, cleanup := mockServer(t, http.MethodGet, "/guilds/111/channels", http.StatusOK, []map[string]any{
		{"id": "100", "name": "general", "type": 0, "position": 0},
		{"id": "200", "name": "voice", "type": 2, "position": 1},
	})
	defer cleanup()

	action := &listChannelsAction{conn: conn}
	params, _ := json.Marshal(listChannelsParams{GuildID: "111"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Channels []listChannelSummary `json:"channels"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(data.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(data.Channels))
	}
	if data.Channels[0].Name != "general" {
		t.Errorf("expected first channel 'general', got %q", data.Channels[0].Name)
	}
}

func TestListChannels_MissingGuildID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing guild_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
