package sqldb

import (
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckTableAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		table   string
		allowed []string
		wantErr bool
	}{
		{"empty_allowlist", "anything", nil, false},
		{"allowed", "users", []string{"users", "orders"}, false},
		{"case_insensitive", "Users", []string{"users"}, false},
		{"not_allowed", "secrets", []string{"users", "orders"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTableAllowed(tt.table, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTableAllowed(%q, %v) error = %v, wantErr %v", tt.table, tt.allowed, err, tt.wantErr)
			}
			if err != nil {
				var ve *connectors.ValidationError
				if !errors.As(err, &ve) {
					t.Errorf("expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestCheckColumnsAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		columns []string
		allowed []string
		wantErr bool
	}{
		{"empty_allowlist", []string{"any"}, nil, false},
		{"all_allowed", []string{"name", "age"}, []string{"name", "age", "email"}, false},
		{"case_insensitive", []string{"Name"}, []string{"name"}, false},
		{"one_disallowed", []string{"name", "secret"}, []string{"name", "age"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckColumnsAllowed(tt.columns, tt.allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckColumnsAllowed(%v, %v) error = %v, wantErr %v", tt.columns, tt.allowed, err, tt.wantErr)
			}
		})
	}
}
