package dynamodb

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDynamoDBConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "dynamodb" {
		t.Errorf("ID() = %q, want %q", got, "dynamodb")
	}
}

func TestDynamoDBConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	want := []string{
		"dynamodb.get_item",
		"dynamodb.put_item",
		"dynamodb.delete_item",
		"dynamodb.query",
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

func TestDynamoDBConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()
	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name: "valid keys",
			creds: connectors.NewCredentials(map[string]string{
				"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
				"secret_access_key": "secret",
			}),
			wantErr: false,
		},
		{
			name:    "missing access key",
			creds:   connectors.NewCredentials(map[string]string{"secret_access_key": "x"}),
			wantErr: true,
		},
		{
			name:    "missing secret",
			creds:   connectors.NewCredentials(map[string]string{"access_key_id": "x"}),
			wantErr: true,
		},
		{
			name: "optional endpoint_url valid",
			creds: connectors.NewCredentials(map[string]string{
				"access_key_id": "AKIA", "secret_access_key": "s",
				"endpoint_url": "http://localhost:4566",
			}),
			wantErr: false,
		},
		{
			name: "invalid endpoint_url",
			creds: connectors.NewCredentials(map[string]string{
				"access_key_id": "AKIA", "secret_access_key": "s",
				"endpoint_url": "not-a-url",
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

func TestDynamoDBConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if m.ID != "dynamodb" {
		t.Errorf("Manifest().ID = %q, want dynamodb", m.ID)
	}
	if len(m.Actions) != 4 {
		t.Fatalf("want 4 actions, got %d", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 || m.RequiredCredentials[0].Service != "dynamodb" {
		t.Fatalf("unexpected credentials: %+v", m.RequiredCredentials)
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestDynamoDBConnector_ActionsMatchManifest(t *testing.T) {
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

func TestDynamoDBConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*DynamoDBConnector)(nil)
	var _ connectors.ManifestProvider = (*DynamoDBConnector)(nil)
}
