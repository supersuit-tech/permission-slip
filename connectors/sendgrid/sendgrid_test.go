package sendgrid

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendGridConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "sendgrid" {
		t.Errorf("ID() = %q, want %q", got, "sendgrid")
	}
}

func TestSendGridConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"sendgrid.send_campaign",
		"sendgrid.schedule_campaign",
		"sendgrid.add_to_list",
		"sendgrid.remove_from_list",
		"sendgrid.create_template",
		"sendgrid.get_campaign_stats",
		"sendgrid.list_segments",
		"sendgrid.list_senders",
		"sendgrid.list_lists",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestSendGridConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid oauth access_token credentials",
			creds:   validOAuthCreds(),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
		{
			name:    "empty access_token falls back to missing api_key",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
		{
			name:    "api_key too short",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "short"}),
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
			t.Parallel()
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

func TestSendGridConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "sendgrid" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "sendgrid")
	}
	if m.Name != "SendGrid" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "SendGrid")
	}
	if len(m.Actions) != 9 {
		t.Fatalf("Manifest().Actions has %d items, want 9", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"sendgrid.send_campaign",
		"sendgrid.schedule_campaign",
		"sendgrid.add_to_list",
		"sendgrid.remove_from_list",
		"sendgrid.create_template",
		"sendgrid.get_campaign_stats",
		"sendgrid.list_segments",
		"sendgrid.list_senders",
		"sendgrid.list_lists",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	// OAuth credential should be first (default in UI).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "sendgrid_oauth" {
		t.Errorf("first credential service = %q, want %q", oauthCred.Service, "sendgrid_oauth")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("first credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "sendgrid" {
		t.Errorf("first credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "sendgrid")
	}
	// API key credential should be second.
	apiKeyCred := m.RequiredCredentials[1]
	if apiKeyCred.Service != "sendgrid" {
		t.Errorf("second credential service = %q, want %q", apiKeyCred.Service, "sendgrid")
	}
	if apiKeyCred.AuthType != "api_key" {
		t.Errorf("second credential auth_type = %q, want %q", apiKeyCred.AuthType, "api_key")
	}
	if apiKeyCred.InstructionsURL == "" {
		t.Error("second credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestSendGridConnector_ActionsMatchManifest(t *testing.T) {
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

func TestSendGridConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SendGridConnector)(nil)
	var _ connectors.ManifestProvider = (*SendGridConnector)(nil)
}
