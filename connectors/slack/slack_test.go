package slack

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSlackConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "slack" {
		t.Errorf("expected ID 'slack', got %q", c.ID())
	}
}

func TestSlackConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"slack.send_message",
		"slack.create_channel",
		"slack.list_channels",
		"slack.read_channel_messages",
		"slack.read_thread",
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("expected action %q to be registered", name)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestSlackConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid bot_token",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "xoxb-1234567890-abcdef"}),
			wantErr: false,
		},
		{
			name:    "missing bot_token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty bot_token",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": ""}),
			wantErr: true,
		},
		{
			name:    "wrong prefix xoxp",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "xoxp-user-token"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix xoxa",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "xoxa-app-token"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix bearer",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "bearer-123"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix sk",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "sk-123"}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestSlackConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "slack" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "slack")
	}
	if m.Name != "Slack" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Slack")
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{"slack.send_message", "slack.create_channel", "slack.list_channels", "slack.read_channel_messages", "slack.read_thread"} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "slack" {
		t.Errorf("credential service = %q, want %q", cred.Service, "slack")
	}
	if cred.AuthType != "custom" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "custom")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestSlackConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestSlackConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SlackConnector)(nil)
	var _ connectors.ManifestProvider = (*SlackConnector)(nil)
}
