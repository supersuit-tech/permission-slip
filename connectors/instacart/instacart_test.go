package instacart

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestInstacartConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "instacart" {
		t.Errorf("ID() = %q, want %q", got, "instacart")
	}
}

func TestInstacartConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	want := []string{"instacart.create_products_link"}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestInstacartConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid with sandbox base_url",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "longenough", "base_url": "https://connect.dev.instacart.tools"}),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "short api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "short"}),
			wantErr: true,
		},
		{
			name:    "disallowed base_url host",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "longenough", "base_url": "https://evil.com"}),
			wantErr: true,
		},
		{
			name:    "http base_url rejected",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "longenough", "base_url": "http://connect.instacart.com"}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstacartConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if m.ID != "instacart" {
		t.Errorf("Manifest().ID = %q, want instacart", m.ID)
	}
	if len(m.Actions) != 1 {
		t.Fatalf("Manifest().Actions has %d items, want 1", len(m.Actions))
	}
	if m.Actions[0].ActionType != "instacart.create_products_link" {
		t.Errorf("unexpected action type %q", m.Actions[0].ActionType)
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}

	manifest := c.Manifest()
	for actionType := range c.Actions() {
		found := false
		for _, a := range manifest.Actions {
			if a.ActionType == actionType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := c.Actions()[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestInstacartConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*InstacartConnector)(nil)
	var _ connectors.ManifestProvider = (*InstacartConnector)(nil)

	a := New().Actions()["instacart.create_products_link"]
	if _, ok := a.(connectors.ParameterAliaser); !ok {
		t.Error("create_products_link should implement ParameterAliaser")
	}
	if _, ok := a.(connectors.Normalizer); !ok {
		t.Error("create_products_link should implement Normalizer")
	}
	if _, ok := a.(connectors.RequestValidator); !ok {
		t.Error("create_products_link should implement RequestValidator")
	}
}
