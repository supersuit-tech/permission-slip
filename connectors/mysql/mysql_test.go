package mysql

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestMySQLConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "mysql" {
		t.Errorf("ID() = %q, want %q", got, "mysql")
	}
}

func TestMySQLConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{"mysql.query", "mysql.insert", "mysql.update", "mysql.delete"}
	for _, actionType := range expected {
		if _, ok := actions[actionType]; !ok {
			t.Errorf("Actions() missing %q", actionType)
		}
	}
	if len(actions) != len(expected) {
		t.Errorf("Actions() has %d actions, want %d", len(actions), len(expected))
	}
}

func TestMySQLConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "mysql" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "mysql")
	}
	if m.Name != "MySQL" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "MySQL")
	}
	if len(m.Actions) != 4 {
		t.Errorf("Manifest().Actions has %d entries, want 4", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 {
		t.Errorf("Manifest().RequiredCredentials has %d entries, want 1", len(m.RequiredCredentials))
	}

	// Validate the manifest passes its own validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestMySQLConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid dsn",
			creds:   connectors.NewCredentials(map[string]string{"dsn": "user:pass@tcp(localhost:3306)/db"}),
			wantErr: false,
		},
		{
			name:    "missing dsn",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty dsn",
			creds:   connectors.NewCredentials(map[string]string{"dsn": ""}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(context.Background(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "users", true},
		{"with underscore", "user_name", true},
		{"with number", "table1", true},
		{"uppercase", "Users", true},
		{"empty", "", false},
		{"starts with number", "1table", false},
		{"has space", "user name", false},
		{"has dash", "user-name", false},
		{"has dot", "db.table", false},
		{"has semicolon", "users;", false},
		{"sql injection", "users; DROP TABLE users", false},
		{"backtick", "`users`", false},
		{"too long", string(make([]byte, 65)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidIdentifier(tt.input); got != tt.want {
				t.Errorf("isValidIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

