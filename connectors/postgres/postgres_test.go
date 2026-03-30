package postgres

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestPostgresConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "postgres" {
		t.Errorf("ID() = %q, want %q", got, "postgres")
	}
}

func TestPostgresConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{"postgres.query", "postgres.insert", "postgres.update", "postgres.delete"}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestPostgresConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid connection_string",
			creds:   connectors.NewCredentials(map[string]string{"connection_string": "postgres://user:pass@localhost:5432/mydb"}),
			wantErr: false,
		},
		{
			name:    "valid postgresql scheme",
			creds:   connectors.NewCredentials(map[string]string{"connection_string": "postgresql://user:pass@localhost/mydb"}),
			wantErr: false,
		},
		{
			name:    "missing connection_string",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty connection_string",
			creds:   connectors.NewCredentials(map[string]string{"connection_string": ""}),
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			creds:   connectors.NewCredentials(map[string]string{"connection_string": "mysql://user:pass@localhost/mydb"}),
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

func TestPostgresConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "postgres" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "postgres")
	}
	if m.Name != "PostgreSQL" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "PostgreSQL")
	}
	if len(m.Actions) != 4 {
		t.Fatalf("Manifest().Actions has %d items, want 4", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{"postgres.query", "postgres.insert", "postgres.update", "postgres.delete"} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "postgres" {
		t.Errorf("credential service = %q, want %q", cred.Service, "postgres")
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

func TestPostgresConnector_ActionsMatchManifest(t *testing.T) {
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

func TestPostgresConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*PostgresConnector)(nil)
	var _ connectors.ManifestProvider = (*PostgresConnector)(nil)
}
