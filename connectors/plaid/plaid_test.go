package plaid

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestPlaidConnector_ID(t *testing.T) {
	t.Parallel()
	conn := New()
	if got := conn.ID(); got != "plaid" {
		t.Errorf("ID() = %q, want %q", got, "plaid")
	}
}

func TestPlaidConnector_Actions(t *testing.T) {
	t.Parallel()
	conn := New()
	actions := conn.Actions()

	expected := []string{
		"plaid.create_link_token",
		"plaid.get_balances",
		"plaid.list_transactions",
		"plaid.get_accounts",
		"plaid.get_identity",
		"plaid.get_institution",
	}
	for _, key := range expected {
		if _, ok := actions[key]; !ok {
			t.Errorf("Actions() missing key %q", key)
		}
	}
	if len(actions) != len(expected) {
		t.Errorf("Actions() has %d entries, want %d", len(actions), len(expected))
	}
}

func TestPlaidConnector_Manifest(t *testing.T) {
	t.Parallel()
	conn := New()
	m := conn.Manifest()

	if m.ID != "plaid" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "plaid")
	}
	if len(m.Actions) != 6 {
		t.Errorf("Manifest().Actions has %d entries, want 6", len(m.Actions))
	}
	if len(m.Templates) != 6 {
		t.Errorf("Manifest().Templates has %d entries, want 6", len(m.Templates))
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestPlaidConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	conn := New()

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
			name:    "missing client_id",
			creds:   connectors.NewCredentials(map[string]string{"secret": testSecret}),
			wantErr: true,
		},
		{
			name:    "missing secret",
			creds:   connectors.NewCredentials(map[string]string{"client_id": testClientID}),
			wantErr: true,
		},
		{
			name:    "empty credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "client_id too short",
			creds:   connectors.NewCredentials(map[string]string{"client_id": "short", "secret": testSecret}),
			wantErr: true,
		},
		{
			name:    "secret too short",
			creds:   connectors.NewCredentials(map[string]string{"client_id": testClientID, "secret": "short"}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
