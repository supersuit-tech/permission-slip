package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestManageRoles_Assign(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT for assign, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/111/members/222/roles/333" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &manageRolesAction{conn: conn}

	params, _ := json.Marshal(manageRolesParams{
		GuildID: "111",
		UserID:  "222",
		RoleID:  "333",
		Action:  "assign",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.manage_roles",
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
	if data["status"] != "success" {
		t.Errorf("expected status 'success', got %q", data["status"])
	}
	if data["action"] != "assign" {
		t.Errorf("expected action 'assign', got %q", data["action"])
	}
}

func TestManageRoles_Remove(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE for remove, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &manageRolesAction{conn: conn}

	params, _ := json.Marshal(manageRolesParams{
		GuildID: "111",
		UserID:  "222",
		RoleID:  "333",
		Action:  "remove",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.manage_roles",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManageRoles_InvalidAction(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &manageRolesAction{conn: conn}

	params, _ := json.Marshal(manageRolesParams{
		GuildID: "111",
		UserID:  "222",
		RoleID:  "333",
		Action:  "invalid",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.manage_roles",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
