package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListRoles_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/111/roles" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": "100", "name": "@everyone", "color": 0, "position": 0, "managed": false, "mentionable": false},
			{"id": "200", "name": "Moderator", "color": 3447003, "position": 1, "managed": false, "mentionable": true},
			{"id": "300", "name": "Bot Role", "color": 0, "position": 2, "managed": true, "mentionable": false},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listRolesAction{conn: conn}

	params, _ := json.Marshal(listRolesParams{GuildID: "111"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.list_roles",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Roles []roleSummary `json:"roles"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(data.Roles) != 3 {
		t.Errorf("expected 3 roles, got %d", len(data.Roles))
	}
	if data.Roles[1].Name != "Moderator" {
		t.Errorf("expected second role 'Moderator', got %q", data.Roles[1].Name)
	}
	if !data.Roles[1].Mentionable {
		t.Error("expected Moderator role to be mentionable")
	}
	if !data.Roles[2].Managed {
		t.Error("expected Bot Role to be managed")
	}
}

func TestListRoles_MissingGuildID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &listRolesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.list_roles",
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
