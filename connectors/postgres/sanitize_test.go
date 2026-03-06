package postgres

import "testing"

func TestValidateIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple", "users", false},
		{"with_underscore", "user_accounts", false},
		{"schema_qualified", "public.users", false},
		{"starts_with_underscore", "_private", false},
		{"empty", "", true},
		{"has_space", "user accounts", true},
		{"has_semicolon", "users;", true},
		{"has_quotes", `users"`, true},
		{"has_dash", "user-accounts", true},
		{"sql_injection", "'; DROP TABLE users--", true},
		{"starts_with_number", "1users", true},
		{"too_long", string(make([]byte, 200)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIdentifier(tt.input, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIdentifier(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"users", `"users"`},
		{"public.users", `"public"."users"`},
		{`has"quote`, `"has""quote"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := quoteIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("quoteIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSortedKeys(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{"c": 3, "a": 1, "b": 2}
	keys := sortedKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("sortedKeys() = %v, want [a b c]", keys)
	}
}
