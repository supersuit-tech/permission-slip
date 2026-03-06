package redis

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestRedisConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "redis" {
		t.Errorf("expected ID 'redis', got %q", c.ID())
	}
}

func TestRedisConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"redis.get", "redis.set", "redis.delete",
		"redis.lpush", "redis.rpush", "redis.lpop", "redis.rpop",
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

func TestRedisConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid redis URL",
			creds:   connectors.NewCredentials(map[string]string{"url": "redis://localhost:6379/0"}),
			wantErr: false,
		},
		{
			name:    "valid rediss URL",
			creds:   connectors.NewCredentials(map[string]string{"url": "rediss://user:pass@redis.example.com:6380/1"}),
			wantErr: false,
		},
		{
			name:    "missing url",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty url",
			creds:   connectors.NewCredentials(map[string]string{"url": ""}),
			wantErr: true,
		},
		{
			name:    "wrong scheme http",
			creds:   connectors.NewCredentials(map[string]string{"url": "http://localhost:6379"}),
			wantErr: true,
		},
		{
			name:    "wrong scheme postgres",
			creds:   connectors.NewCredentials(map[string]string{"url": "postgres://localhost:5432"}),
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

func TestRedisConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "redis" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "redis")
	}
	if m.Name != "Redis" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Redis")
	}
	if len(m.Actions) != 7 {
		t.Fatalf("Manifest().Actions has %d items, want 7", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"redis.get", "redis.set", "redis.delete",
		"redis.lpush", "redis.rpush", "redis.lpop", "redis.rpop",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "redis" {
		t.Errorf("credential service = %q, want %q", cred.Service, "redis")
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

func TestRedisConnector_ActionsMatchManifest(t *testing.T) {
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

func TestRedisConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*RedisConnector)(nil)
	var _ connectors.ManifestProvider = (*RedisConnector)(nil)
}
