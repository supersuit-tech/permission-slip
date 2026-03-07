package protonmail

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestProtonMailConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "protonmail" {
		t.Errorf("expected ID 'protonmail', got %q", c.ID())
	}
}

func TestProtonMailConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"protonmail.send_email",
		"protonmail.read_inbox",
		"protonmail.search_emails",
		"protonmail.read_email",
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

func TestProtonMailConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid with all fields",
			creds:   validCredsAllFields(),
			wantErr: false,
		},
		{
			name:    "missing username",
			creds:   connectors.NewCredentials(map[string]string{"password": "bridge-pass"}),
			wantErr: true,
		},
		{
			name:    "missing password",
			creds:   connectors.NewCredentials(map[string]string{"username": "user@proton.me"}),
			wantErr: true,
		},
		{
			name:    "empty username",
			creds:   connectors.NewCredentials(map[string]string{"username": "", "password": "bridge-pass"}),
			wantErr: true,
		},
		{
			name:    "empty password",
			creds:   connectors.NewCredentials(map[string]string{"username": "user@proton.me", "password": ""}),
			wantErr: true,
		},
		{
			name:    "empty credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
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

func TestProtonMailConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "protonmail" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "protonmail")
	}
	if m.Name != "Proton Mail" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Proton Mail")
	}
	if len(m.Actions) != 4 {
		t.Fatalf("Manifest().Actions has %d items, want 4", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"protonmail.send_email",
		"protonmail.read_inbox",
		"protonmail.search_emails",
		"protonmail.read_email",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "protonmail" {
		t.Errorf("credential service = %q, want %q", cred.Service, "protonmail")
	}
	if cred.AuthType != "custom" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "custom")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	// Verify send_email is high risk.
	for _, a := range m.Actions {
		if a.ActionType == "protonmail.send_email" && a.RiskLevel != "high" {
			t.Errorf("send_email risk_level = %q, want %q", a.RiskLevel, "high")
		}
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestProtonMailConnector_ActionsMatchManifest(t *testing.T) {
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

func TestProtonMailConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ProtonMailConnector)(nil)
	var _ connectors.ManifestProvider = (*ProtonMailConnector)(nil)
}
