package zapier

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestZapierConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "zapier" {
		t.Errorf("expected ID 'zapier', got %q", c.ID())
	}
}

func TestZapierConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"zapier.trigger_webhook",
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

func TestZapierConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid webhook URL",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing webhook_url",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty webhook_url",
			creds:   connectors.NewCredentials(map[string]string{"webhook_url": ""}),
			wantErr: true,
		},
		{
			name:    "non-https URL",
			creds:   connectors.NewCredentials(map[string]string{"webhook_url": "http://hooks.zapier.com/hooks/catch/123/abc/"}),
			wantErr: true,
		},
		{
			name:    "invalid URL",
			creds:   connectors.NewCredentials(map[string]string{"webhook_url": "://not-a-url"}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
		{
			name:    "custom domain HTTPS is valid",
			creds:   connectors.NewCredentials(map[string]string{"webhook_url": "https://custom-webhook.example.com/hook"}),
			wantErr: false,
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

func TestZapierConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "zapier" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "zapier")
	}
	if m.Name != "Zapier" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Zapier")
	}
	if len(m.Actions) != 1 {
		t.Fatalf("Manifest().Actions has %d items, want 1", len(m.Actions))
	}
	if m.Actions[0].ActionType != "zapier.trigger_webhook" {
		t.Errorf("Manifest().Actions[0].ActionType = %q, want %q", m.Actions[0].ActionType, "zapier.trigger_webhook")
	}
	if m.Actions[0].RiskLevel != "high" {
		t.Errorf("Manifest().Actions[0].RiskLevel = %q, want %q", m.Actions[0].RiskLevel, "high")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "zapier" {
		t.Errorf("credential service = %q, want %q", cred.Service, "zapier")
	}
	if cred.AuthType != "custom" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "custom")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestZapierConnector_ActionsMatchManifest(t *testing.T) {
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

func TestZapierConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ZapierConnector)(nil)
	var _ connectors.ManifestProvider = (*ZapierConnector)(nil)
}
