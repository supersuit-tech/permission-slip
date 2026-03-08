package monday

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMondayConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "monday" {
		t.Errorf("expected ID 'monday', got %q", c.ID())
	}
}

func TestMondayConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"monday.create_item",
		"monday.update_item",
		"monday.add_update",
		"monday.create_subitem",
		"monday.move_item_to_group",
		"monday.search_items",
		"monday.list_boards",
		"monday.get_board",
		"monday.create_board",
		"monday.delete_item",
		"monday.list_groups",
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

func TestMondayConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "eyJhbGciOiJIUzI1NiJ9.test"}),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
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

func TestMondayConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "monday" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "monday")
	}
	if m.Name != "Monday.com" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Monday.com")
	}
	if len(m.Actions) != 11 {
		t.Fatalf("Manifest().Actions has %d items, want 11", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"monday.create_item",
		"monday.update_item",
		"monday.add_update",
		"monday.create_subitem",
		"monday.move_item_to_group",
		"monday.search_items",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "monday" {
		t.Errorf("credential service = %q, want %q", cred.Service, "monday")
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

func TestMondayConnector_ActionsMatchManifest(t *testing.T) {
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

func TestMondayConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*MondayConnector)(nil)
	var _ connectors.ManifestProvider = (*MondayConnector)(nil)
}

func TestIsValidMondayID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id   string
		want bool
	}{
		{"12345", true},
		{"0", true},
		{"9876543210", true},
		{"", false},
		{"abc", false},
		{"123abc", false},
		{"12.34", false},
		{"-1", false},
		{"12 34", false},
	}
	for _, tt := range tests {
		if got := isValidMondayID(tt.id); got != tt.want {
			t.Errorf("isValidMondayID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
