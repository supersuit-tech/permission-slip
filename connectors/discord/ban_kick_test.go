package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestBanUser_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/111/bans/222" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &banUserAction{conn: conn}

	params, _ := json.Marshal(banUserParams{
		GuildID: "111",
		UserID:  "222",
	})

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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/111/members/222" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &kickUserAction{conn: conn}

	params, _ := json.Marshal(kickUserParams{
		GuildID: "111",
		UserID:  "222",
	})

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
