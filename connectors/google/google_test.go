package google

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGoogleConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "google" {
		t.Errorf("expected ID 'google', got %q", c.ID())
	}
}

func TestGoogleConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"google.send_email",
		"google.list_emails",
		"google.create_calendar_event",
		"google.list_calendar_events",
		"google.send_chat_message",
		"google.list_chat_spaces",
		"google.create_meeting",
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

func TestGoogleConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "ya29.test-token"}),
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

func TestGoogleConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "google" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "google")
	}
	if m.Name != "Google" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Google")
	}
	if len(m.Actions) != 7 {
		t.Fatalf("Manifest().Actions has %d items, want 7", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"google.send_email",
		"google.list_emails",
		"google.create_calendar_event",
		"google.list_calendar_events",
		"google.send_chat_message",
		"google.list_chat_spaces",
		"google.create_meeting",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "google" {
		t.Errorf("credential service = %q, want %q", cred.Service, "google")
	}
	if cred.AuthType != "oauth2" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "oauth2")
	}
	if cred.OAuthProvider != "google" {
		t.Errorf("credential oauth_provider = %q, want %q", cred.OAuthProvider, "google")
	}
	if len(cred.OAuthScopes) == 0 {
		t.Error("credential oauth_scopes is empty, want at least one scope")
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestGoogleConnector_ActionsMatchManifest(t *testing.T) {
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

func TestGoogleConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*GoogleConnector)(nil)
	var _ connectors.ManifestProvider = (*GoogleConnector)(nil)
}
