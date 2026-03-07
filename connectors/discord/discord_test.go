package discord

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDiscordConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "discord" {
		t.Errorf("expected ID 'discord', got %q", c.ID())
	}
}

func TestDiscordConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}
	if m.ID != "discord" {
		t.Errorf("expected manifest ID 'discord', got %q", m.ID)
	}
	if err := m.Validate(); err != nil {
		t.Errorf("manifest validation failed: %v", err)
	}
}

func TestDiscordConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expectedActions := []string{
		"discord.send_message",
		"discord.create_channel",
		"discord.manage_roles",
		"discord.create_event",
		"discord.ban_user",
		"discord.kick_user",
		"discord.pin_message",
		"discord.unpin_message",
		"discord.list_channels",
		"discord.list_members",
		"discord.create_thread",
		"discord.list_roles",
	}

	for _, actionType := range expectedActions {
		if _, ok := actions[actionType]; !ok {
			t.Errorf("missing action: %s", actionType)
		}
	}

	if len(actions) != len(expectedActions) {
		t.Errorf("expected %d actions, got %d", len(expectedActions), len(actions))
	}
}

func TestDiscordConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid token",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "test-token"}),
			wantErr: false,
		},
		{
			name:    "missing token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty token",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": ""}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSnowflake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "123456789012345678", false},
		{"empty", "", true},
		{"non-numeric", "abc123", true},
		{"has spaces", "123 456", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSnowflake(tt.value, "test_id")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSnowflake(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestManifestToDBConversion(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	dbManifest := m.ToDBManifest()
	if dbManifest.ID != "discord" {
		t.Errorf("expected DB manifest ID 'discord', got %q", dbManifest.ID)
	}
	if len(dbManifest.Actions) != 12 {
		t.Errorf("expected 12 DB actions, got %d", len(dbManifest.Actions))
	}
}

func TestMapDiscordError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		code       int
		message    string
		wantType   string
		wantSubstr string
	}{
		{
			name:       "unknown channel",
			statusCode: 404,
			code:       10003,
			message:    "Unknown Channel",
			wantType:   "external",
			wantSubstr: "channel not found",
		},
		{
			name:       "missing permissions",
			statusCode: 403,
			code:       50013,
			message:    "Missing Permissions",
			wantType:   "auth",
			wantSubstr: "missing a required permission",
		},
		{
			name:       "unknown role",
			statusCode: 404,
			code:       10011,
			message:    "Unknown Role",
			wantType:   "external",
			wantSubstr: "list_roles",
		},
		{
			name:       "max pins",
			statusCode: 400,
			code:       30003,
			message:    "Maximum number of pins reached",
			wantType:   "external",
			wantSubstr: "50 pinned messages",
		},
		{
			name:       "unauthorized fallback",
			statusCode: 401,
			code:       0,
			message:    "",
			wantType:   "auth",
			wantSubstr: "invalid bot token",
		},
		{
			name:       "generic error",
			statusCode: 500,
			code:       0,
			message:    "Internal Server Error",
			wantType:   "external",
			wantSubstr: "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := mapDiscordError(tt.statusCode, discordErrorResponse{Code: tt.code, Message: tt.message})
			if err == nil {
				t.Fatal("expected error")
			}
			switch tt.wantType {
			case "auth":
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			case "external":
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
				}
			case "validation":
				if !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T: %v", err, err)
				}
			}
			if got := err.Error(); !containsSubstring(got, tt.wantSubstr) {
				t.Errorf("error %q should contain %q", got, tt.wantSubstr)
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (sub == "" || contains(s, sub))
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestManifestTemplateParametersAreValidJSON(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	for _, tpl := range m.Templates {
		var params map[string]json.RawMessage
		if err := json.Unmarshal(tpl.Parameters, &params); err != nil {
			t.Errorf("template %q has invalid parameters JSON: %v", tpl.ID, err)
		}
	}
}
