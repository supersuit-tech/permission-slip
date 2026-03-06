package microsoft

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMicrosoftConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "microsoft" {
		t.Errorf("expected ID 'microsoft', got %q", c.ID())
	}
}

func TestMicrosoftConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"microsoft.send_email",
		"microsoft.list_emails",
		"microsoft.create_calendar_event",
		"microsoft.list_calendar_events",
		"microsoft.create_document",
		"microsoft.get_document",
		"microsoft.update_document",
		"microsoft.list_documents",
		"microsoft.list_teams",
		"microsoft.list_channels",
		"microsoft.send_channel_message",
		"microsoft.list_channel_messages",
		"microsoft.create_presentation",
		"microsoft.list_presentations",
		"microsoft.get_presentation",
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

func TestMicrosoftConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "test-token-123"}),
			wantErr: false,
		},
		{
			name:    "missing access_token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
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

func TestMicrosoftConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "microsoft" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "microsoft")
	}
	if m.Name != "Microsoft" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Microsoft")
	}
	if len(m.Actions) != 15 {
		t.Fatalf("Manifest().Actions has %d items, want 15", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"microsoft.send_email",
		"microsoft.list_emails",
		"microsoft.create_calendar_event",
		"microsoft.list_calendar_events",
		"microsoft.create_document",
		"microsoft.get_document",
		"microsoft.update_document",
		"microsoft.list_documents",
		"microsoft.list_teams",
		"microsoft.list_channels",
		"microsoft.send_channel_message",
		"microsoft.list_channel_messages",
		"microsoft.create_presentation",
		"microsoft.list_presentations",
		"microsoft.get_presentation",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "microsoft" {
		t.Errorf("credential service = %q, want %q", cred.Service, "microsoft")
	}
	if cred.AuthType != "oauth2" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "oauth2")
	}
	if cred.OAuthProvider != "microsoft" {
		t.Errorf("credential oauth_provider = %q, want %q", cred.OAuthProvider, "microsoft")
	}
	if len(cred.OAuthScopes) == 0 {
		t.Error("credential oauth_scopes is empty, want at least one scope")
	}

	// Validate templates.
	if len(m.Templates) != 15 {
		t.Errorf("Manifest().Templates has %d items, want 15", len(m.Templates))
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestMicrosoftConnector_ActionsMatchManifest(t *testing.T) {
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

func TestMicrosoftConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*MicrosoftConnector)(nil)
	var _ connectors.ManifestProvider = (*MicrosoftConnector)(nil)
}
