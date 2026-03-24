package firestore

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestFirestoreConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "firestore" {
		t.Errorf("ID() = %q, want %q", got, "firestore")
	}
}

func TestFirestoreConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	want := []string{
		"firestore.get",
		"firestore.set",
		"firestore.update",
		"firestore.delete",
		"firestore.query",
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

func TestFirestoreConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()
	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid SA JSON with project_id in JSON",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name: "valid with explicit project_id credential",
			creds: connectors.NewCredentials(map[string]string{
				"service_account_json": `{"type":"service_account","client_email":"a@b.com","private_key":"k","project_id":""}`,
				"project_id":           "override-proj",
			}),
			wantErr: false,
		},
		{
			name:    "missing JSON",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			creds:   connectors.NewCredentials(map[string]string{"service_account_json": "not-json"}),
			wantErr: true,
		},
		{
			name: "missing project_id",
			creds: connectors.NewCredentials(map[string]string{
				"service_account_json": `{"type":"service_account","client_email":"a@b.com","private_key":"k"}`,
			}),
			wantErr: true,
		},
		{
			name: "bad emulator host",
			creds: connectors.NewCredentials(map[string]string{
				"service_account_json": validServiceAccountJSON(),
				"emulator_host":        "http://bad",
			}),
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
				t.Errorf("got %T, want validation error", err)
			}
		})
	}
}

func TestFirestoreConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if m.ID != "firestore" {
		t.Errorf("Manifest().ID = %q, want firestore", m.ID)
	}
	if len(m.Actions) != 5 {
		t.Fatalf("want 5 actions, got %d", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 || m.RequiredCredentials[0].Service != "firestore" {
		t.Fatalf("unexpected credentials: %+v", m.RequiredCredentials)
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestFirestoreConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()
	seen := make(map[string]bool)
	for _, a := range manifest.Actions {
		seen[a.ActionType] = true
	}
	for at := range actions {
		if !seen[at] {
			t.Errorf("action %q not in manifest", at)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("manifest action %q not registered", a.ActionType)
		}
	}
}

func TestFirestoreConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*FirestoreConnector)(nil)
	var _ connectors.ManifestProvider = (*FirestoreConnector)(nil)
}
