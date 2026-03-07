package datadog

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDatadogConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "datadog" {
		t.Errorf("ID() = %q, want %q", got, "datadog")
	}
}

func TestDatadogConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"datadog.get_metrics",
		"datadog.create_incident",
		"datadog.snooze_alert",
		"datadog.trigger_runbook",
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

func TestDatadogConnector_ValidateCredentials(t *testing.T) {
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
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{"app_key": "test"}),
			wantErr: true,
		},
		{
			name:    "missing app_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test"}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "", "app_key": "test"}),
			wantErr: true,
		},
		{
			name:    "empty app_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test", "app_key": ""}),
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

func TestDatadogConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "datadog" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "datadog")
	}
	if m.Name != "Datadog" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Datadog")
	}
	if len(m.Actions) != 4 {
		t.Fatalf("Manifest().Actions has %d items, want 4", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"datadog.get_metrics",
		"datadog.create_incident",
		"datadog.snooze_alert",
		"datadog.trigger_runbook",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "datadog" {
		t.Errorf("credential service = %q, want %q", cred.Service, "datadog")
	}
	if cred.AuthType != "custom" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "custom")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestDatadogConnector_ActionsMatchManifest(t *testing.T) {
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

func TestDatadogConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*DatadogConnector)(nil)
	var _ connectors.ManifestProvider = (*DatadogConnector)(nil)
}
