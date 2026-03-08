package airtable

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAirtableConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "airtable" {
		t.Errorf("expected ID 'airtable', got %q", c.ID())
	}
}

func TestAirtableConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"airtable.list_bases",
		"airtable.list_records",
		"airtable.get_record",
		"airtable.create_records",
		"airtable.update_records",
		"airtable.delete_records",
		"airtable.search_records",
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

func TestAirtableConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid personal access token",
			creds:   connectors.NewCredentials(map[string]string{"api_token": "patABC123.xyz"}),
			wantErr: false,
		},
		{
			name:    "valid OAuth access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "oauth-test-token-123"}),
			wantErr: false,
		},
		{
			name:    "OAuth access_token takes priority over api_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "oauth-token", "api_token": "patABC123"}),
			wantErr: false,
		},
		{
			name:    "missing both tokens",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_token",
			creds:   connectors.NewCredentials(map[string]string{"api_token": ""}),
			wantErr: true,
		},
		{
			name:    "empty access_token falls back to api_token check",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "", "api_token": "patABC123"}),
			wantErr: false,
		},
		{
			name:    "wrong prefix sk",
			creds:   connectors.NewCredentials(map[string]string{"api_token": "sk-test-123"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix key",
			creds:   connectors.NewCredentials(map[string]string{"api_token": "key12345"}),
			wantErr: true,
		},
		{
			name:    "token with unsafe characters",
			creds:   connectors.NewCredentials(map[string]string{"api_token": "pat\x00injected"}),
			wantErr: true,
		},
		{
			name:    "token with spaces",
			creds:   connectors.NewCredentials(map[string]string{"api_token": "pat token with spaces"}),
			wantErr: true,
		},
		{
			name:    "OAuth token with unsafe characters",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "token\x00injected"}),
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

func TestAirtableConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "airtable" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "airtable")
	}
	if m.Name != "Airtable" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Airtable")
	}
	if len(m.Actions) != 7 {
		t.Fatalf("Manifest().Actions has %d items, want 7", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"airtable.list_bases",
		"airtable.list_records",
		"airtable.get_record",
		"airtable.create_records",
		"airtable.update_records",
		"airtable.delete_records",
		"airtable.search_records",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	// First credential should be OAuth (primary/recommended).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "airtable" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "airtable")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "airtable" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "airtable")
	}
	if len(oauthCred.OAuthScopes) == 0 {
		t.Error("oauth credential oauth_scopes is empty, want scopes")
	}
	// Second credential should be API key (alternative).
	apiKeyCred := m.RequiredCredentials[1]
	if apiKeyCred.Service != "airtable" {
		t.Errorf("api_key credential service = %q, want %q", apiKeyCred.Service, "airtable")
	}
	if apiKeyCred.AuthType != "api_key" {
		t.Errorf("api_key credential auth_type = %q, want %q", apiKeyCred.AuthType, "api_key")
	}
	if apiKeyCred.InstructionsURL == "" {
		t.Error("api_key credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestAirtableConnector_ActionsMatchManifest(t *testing.T) {
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

func TestAirtableConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*AirtableConnector)(nil)
	var _ connectors.ManifestProvider = (*AirtableConnector)(nil)
}
