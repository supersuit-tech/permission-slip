package supabase

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSupabaseConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "supabase" {
		t.Errorf("expected ID 'supabase', got %q", c.ID())
	}
}

func TestSupabaseConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"supabase.read",
		"supabase.insert",
		"supabase.update",
		"supabase.delete",
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

func TestSupabaseConnector_ValidateCredentials(t *testing.T) {
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
			name:    "valid credentials with https URL",
			creds:   connectors.NewCredentials(map[string]string{"supabase_url": "https://abc.supabase.co", "api_key": "test-key"}),
			wantErr: false,
		},
		{
			name:    "missing supabase_url",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test-key"}),
			wantErr: true,
		},
		{
			name:    "empty supabase_url",
			creds:   connectors.NewCredentials(map[string]string{"supabase_url": "", "api_key": "test-key"}),
			wantErr: true,
		},
		{
			name:    "invalid URL scheme",
			creds:   connectors.NewCredentials(map[string]string{"supabase_url": "ftp://bad.example.com", "api_key": "test-key"}),
			wantErr: true,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{"supabase_url": "https://abc.supabase.co"}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"supabase_url": "https://abc.supabase.co", "api_key": ""}),
			wantErr: true,
		},
		{
			name:    "api_key with control characters",
			creds:   connectors.NewCredentials(map[string]string{"supabase_url": "https://abc.supabase.co", "api_key": "key\x00injected"}),
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

func TestSupabaseConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "supabase" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "supabase")
	}
	if m.Name != "Supabase" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Supabase")
	}
	if len(m.Actions) != 4 {
		t.Fatalf("Manifest().Actions has %d items, want 4", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"supabase.read",
		"supabase.insert",
		"supabase.update",
		"supabase.delete",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "supabase" {
		t.Errorf("credential service = %q, want %q", cred.Service, "supabase")
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

func TestSupabaseConnector_ActionsMatchManifest(t *testing.T) {
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

func TestSupabaseConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SupabaseConnector)(nil)
	var _ connectors.ManifestProvider = (*SupabaseConnector)(nil)
}
