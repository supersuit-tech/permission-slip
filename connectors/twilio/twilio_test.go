package twilio

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestTwilioConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "twilio" {
		t.Errorf("ID() = %q, want %q", got, "twilio")
	}
}

func TestTwilioConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"twilio.send_sms",
		"twilio.send_whatsapp",
		"twilio.initiate_call",
		"twilio.get_message",
		"twilio.get_call",
		"twilio.lookup_phone",
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

func TestTwilioConnector_ValidateCredentials(t *testing.T) {
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
			name:    "missing account_sid",
			creds:   connectors.NewCredentials(map[string]string{"auth_token": "tok123"}),
			wantErr: true,
		},
		{
			name:    "empty account_sid",
			creds:   connectors.NewCredentials(map[string]string{"account_sid": "", "auth_token": "tok123"}),
			wantErr: true,
		},
		{
			name:    "invalid account_sid prefix",
			creds:   connectors.NewCredentials(map[string]string{"account_sid": "XX12345678901234567890123456789012", "auth_token": "tok123"}),
			wantErr: true,
		},
		{
			name:    "account_sid too short",
			creds:   connectors.NewCredentials(map[string]string{"account_sid": "AC123", "auth_token": "tok123"}),
			wantErr: true,
		},
		{
			name:    "missing auth_token",
			creds:   connectors.NewCredentials(map[string]string{"account_sid": testAccountSID}),
			wantErr: true,
		},
		{
			name:    "empty auth_token",
			creds:   connectors.NewCredentials(map[string]string{"account_sid": testAccountSID, "auth_token": ""}),
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

func TestTwilioConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "twilio" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "twilio")
	}
	if m.Name != "Twilio" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Twilio")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"twilio.send_sms",
		"twilio.send_whatsapp",
		"twilio.initiate_call",
		"twilio.get_message",
		"twilio.get_call",
		"twilio.lookup_phone",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "twilio" {
		t.Errorf("credential service = %q, want %q", cred.Service, "twilio")
	}
	if cred.AuthType != "basic" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "basic")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestTwilioConnector_ActionsMatchManifest(t *testing.T) {
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

func TestTwilioConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*TwilioConnector)(nil)
	var _ connectors.ManifestProvider = (*TwilioConnector)(nil)
}
