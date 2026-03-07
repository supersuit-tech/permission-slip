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
			name:    "missing api_token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_token",
			creds:   connectors.NewCredentials(map[string]string{"api_token": ""}),
			wantErr: true,
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
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "airtable" {
		t.Errorf("credential service = %q, want %q", cred.Service, "airtable")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
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
