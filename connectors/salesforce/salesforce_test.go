package salesforce

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSalesforceConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "salesforce" {
		t.Errorf("expected ID 'salesforce', got %q", c.ID())
	}
}

func TestSalesforceConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"salesforce.create_record",
		"salesforce.update_record",
		"salesforce.query",
		"salesforce.create_task",
		"salesforce.add_note",
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

func TestSalesforceConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": "https://myorg.salesforce.com"}),
			wantErr: false,
		},
		{
			name:    "missing access_token",
			creds:   connectors.NewCredentials(map[string]string{"instance_url": "https://myorg.salesforce.com"}),
			wantErr: true,
		},
		{
			name:    "missing instance_url",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok"}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "", "instance_url": "https://myorg.salesforce.com"}),
			wantErr: true,
		},
		{
			name:    "empty instance_url",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": ""}),
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

func TestSalesforceConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "salesforce" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "salesforce")
	}
	if m.Name != "Salesforce" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Salesforce")
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"salesforce.create_record",
		"salesforce.update_record",
		"salesforce.query",
		"salesforce.create_task",
		"salesforce.add_note",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "salesforce" {
		t.Errorf("credential service = %q, want %q", cred.Service, "salesforce")
	}
	if cred.AuthType != "oauth2" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "oauth2")
	}
	if cred.OAuthProvider != "salesforce" {
		t.Errorf("credential oauth_provider = %q, want %q", cred.OAuthProvider, "salesforce")
	}
	if len(cred.OAuthScopes) == 0 {
		t.Error("credential oauth_scopes is empty, want at least one scope")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestSalesforceConnector_ActionsMatchManifest(t *testing.T) {
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

func TestSalesforceConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SalesforceConnector)(nil)
	var _ connectors.ManifestProvider = (*SalesforceConnector)(nil)
}
