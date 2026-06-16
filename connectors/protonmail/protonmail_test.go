package protonmail

import (
	"fmt"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

var _ connectors.ManifestProvider = (*ProtonMailConnector)(nil)

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
		"protonmail.reply_email",
		"protonmail.read_inbox",
		"protonmail.search_emails",
		"protonmail.read_email",
		"protonmail.download_attachment",
		"protonmail.archive_email",
		"protonmail.list_folders",
		"protonmail.mark_read",
		"protonmail.mark_unread",
		"protonmail.flag",
		"protonmail.unflag",
		"protonmail.move_to_folder",
		"protonmail.delete",
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

func TestValidateCredentialShape(t *testing.T) {
	t.Parallel()

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
		{
			name:    "invalid smtp_port",
			creds:   connectors.NewCredentials(map[string]string{"username": "user@proton.me", "password": "pass", "smtp_port": "abc"}),
			wantErr: true,
		},
		{
			name:    "invalid imap_port",
			creds:   connectors.NewCredentials(map[string]string{"username": "user@proton.me", "password": "pass", "imap_port": "xyz"}),
			wantErr: true,
		},
		{
			name:    "valid custom ports",
			creds:   connectors.NewCredentials(map[string]string{"username": "user@proton.me", "password": "pass", "smtp_port": "587", "imap_port": "993"}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCredentialShape(tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCredentialShape() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("validateCredentialShape() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestProtonMailConnector_ValidateCredentials(t *testing.T) {
	oldIMAP := testIMAPConn
	oldSMTP := testSMTPConn
	testIMAPConn = func(_ connectors.Credentials, _ time.Duration) error { return nil }
	testSMTPConn = func(_ connectors.Credentials, _ time.Duration) error { return nil }
	t.Cleanup(func() {
		testIMAPConn = oldIMAP
		testSMTPConn = oldSMTP
	})

	c := New()
	for _, creds := range []connectors.Credentials{validCreds(), validCredsAllFields()} {
		if err := c.ValidateCredentials(t.Context(), creds); err != nil {
			t.Errorf("ValidateCredentials() error = %v, want nil", err)
		}
	}
}

func TestProtonMailConnector_ValidateCredentials_proxyUnreachable(t *testing.T) {
	oldIMAP := testIMAPConn
	testIMAPConn = func(creds connectors.Credentials, timeout time.Duration) error {
		session, err := connectIMAP(creds, timeout)
		if err != nil {
			return err
		}
		session.close()
		return nil
	}
	t.Cleanup(func() { testIMAPConn = oldIMAP })

	c := New()
	err := c.ValidateCredentials(t.Context(), validCreds())
	if err == nil {
		t.Fatal("expected error when the local Proton proxy (Bridge or hydroxide) is not running")
	}
	if connectors.IsValidationError(err) {
		t.Fatalf("expected connection/auth error, got validation error: %v", err)
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
	expectedActions := []string{
		"protonmail.send_email",
		"protonmail.reply_email",
		"protonmail.read_inbox",
		"protonmail.search_emails",
		"protonmail.read_email",
		"protonmail.download_attachment",
		"protonmail.archive_email",
		"protonmail.list_folders",
		"protonmail.mark_read",
		"protonmail.mark_unread",
		"protonmail.flag",
		"protonmail.unflag",
		"protonmail.move_to_folder",
		"protonmail.delete",
	}
	if len(m.Actions) != len(expectedActions) {
		t.Fatalf("Manifest().Actions has %d items, want %d", len(m.Actions), len(expectedActions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range expectedActions {
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

	wantFields := []struct {
		key      string
		secret   bool
		required bool
	}{
		{key: "username", secret: false, required: true},
		{key: "password", secret: true, required: true},
		{key: "imap_host", secret: false, required: false},
		{key: "imap_port", secret: false, required: false},
		{key: "smtp_host", secret: false, required: false},
		{key: "smtp_port", secret: false, required: false},
	}
	if len(cred.Fields) != len(wantFields) {
		t.Fatalf("credential fields has %d items, want %d", len(cred.Fields), len(wantFields))
	}
	for i, want := range wantFields {
		got := cred.Fields[i]
		if got.Key != want.key {
			t.Errorf("credential fields[%d].key = %q, want %q", i, got.Key, want.key)
		}
		if got.Label == "" {
			t.Errorf("credential fields[%d].label is empty, want a label", i)
		}
		if got.Secret == nil || *got.Secret != want.secret {
			gotSecret := "<nil>"
			if got.Secret != nil {
				gotSecret = fmt.Sprintf("%v", *got.Secret)
			}
			t.Errorf("credential fields[%d].secret = %s, want %v", i, gotSecret, want.secret)
		}
		if got.Required == nil || *got.Required != want.required {
			gotRequired := "<nil>"
			if got.Required != nil {
				gotRequired = fmt.Sprintf("%v", *got.Required)
			}
			t.Errorf("credential fields[%d].required = %s, want %v", i, gotRequired, want.required)
		}
	}
	for _, key := range []string{"username", "password"} {
		var field connectors.ManifestCredentialField
		for _, f := range cred.Fields {
			if f.Key == key {
				field = f
				break
			}
		}
		if field.HelpText == "" {
			t.Errorf("%s field help_text is empty, want Bridge setup guidance", key)
		}
	}

	for _, a := range m.Actions {
		switch a.ActionType {
		case "protonmail.send_email":
			if a.RiskLevel != "high" {
				t.Errorf("send_email risk_level = %q, want high", a.RiskLevel)
			}
		case "protonmail.reply_email", "protonmail.archive_email":
			if a.RiskLevel != "medium" {
				t.Errorf("%s risk_level = %q, want medium", a.ActionType, a.RiskLevel)
			}
		case "protonmail.read_inbox", "protonmail.search_emails", "protonmail.read_email", "protonmail.download_attachment":
			if a.RiskLevel != "low" {
				t.Errorf("%s risk_level = %q, want low", a.ActionType, a.RiskLevel)
			}
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
