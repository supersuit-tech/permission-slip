package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateChannel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/999888777666555444/channels" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body createChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Name != "announcements" {
			t.Errorf("expected name 'announcements', got %q", body.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "111222333444555666",
			"name": "announcements",
			"type": 0,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		GuildID: "999888777666555444",
		Name:    "announcements",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "111222333444555666" {
		t.Errorf("expected id '111222333444555666', got %v", data["id"])
	}
}

func TestCreateChannel_MissingName(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"guild_id": "999888777666555444"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateChannel_NameInvalidFormat(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createChannelAction{conn: conn}

	tests := []struct {
		name string
		val  string
	}{
		{"uppercase", "General"},
		{"spaces", "my channel"},
		{"special chars", "hello!world"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(createChannelParams{
				GuildID: "999888777666555444",
				Name:    tt.val,
			})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "discord.create_channel",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatalf("expected error for name %q", tt.val)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError for name %q, got: %T", tt.val, err)
			}
		})
	}
}

func TestCreateChannel_TopicTooLong(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createChannelAction{conn: conn}

	longTopic := make([]byte, 1025)
	for i := range longTopic {
		longTopic[i] = 'a'
	}
	params, _ := json.Marshal(createChannelParams{
		GuildID: "999888777666555444",
		Name:    "general",
		Topic:   string(longTopic),
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for topic too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateChannel_InvalidParentID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		GuildID:  "999888777666555444",
		Name:     "general",
		ParentID: "not-a-snowflake",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid parent_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateChannel_NameTooShort(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		GuildID: "999888777666555444",
		Name:    "a",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for name too short")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
