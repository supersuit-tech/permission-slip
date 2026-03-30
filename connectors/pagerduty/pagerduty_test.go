package pagerduty

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestPagerDutyConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "pagerduty" {
		t.Errorf("ID() = %q, want %q", got, "pagerduty")
	}
}

func TestPagerDutyConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"pagerduty.create_incident",
		"pagerduty.acknowledge_alert",
		"pagerduty.resolve_incident",
		"pagerduty.escalate_incident",
		"pagerduty.list_on_call",
		"pagerduty.add_note",
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

func TestPagerDutyConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid access_token (OAuth)",
			creds:   validOAuthCreds(),
			wantErr: false,
		},
		{
			name:    "missing both api_key and access_token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key and no access_token",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
		{
			name:    "empty access_token and no api_key",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
		{
			name:    "wrong key name",
			creds:   connectors.NewCredentials(map[string]string{"token": "pd_test"}),
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

func TestPagerDutyConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "pagerduty" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "pagerduty")
	}
	if m.Name != "PagerDuty" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "PagerDuty")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"pagerduty.create_incident",
		"pagerduty.acknowledge_alert",
		"pagerduty.resolve_incident",
		"pagerduty.escalate_incident",
		"pagerduty.list_on_call",
		"pagerduty.add_note",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}

	// First credential should be OAuth (default/recommended).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "pagerduty" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "pagerduty")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "pagerduty" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "pagerduty")
	}

	// Second credential should be API key (alternative).
	apiKeyCred := m.RequiredCredentials[1]
	if apiKeyCred.Service != "pagerduty" {
		t.Errorf("api_key credential service = %q, want %q", apiKeyCred.Service, "pagerduty")
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

func TestPagerDutyConnector_ActionsMatchManifest(t *testing.T) {
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

func TestPagerDutyConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*PagerDutyConnector)(nil)
	var _ connectors.ManifestProvider = (*PagerDutyConnector)(nil)
}
