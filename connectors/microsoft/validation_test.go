package microsoft

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestValidateGraphID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid id", value: "abc-123", wantErr: false},
		{name: "valid guid", value: "{00000000-0000-0000-0000-000000000000}", wantErr: false},
		{name: "empty", value: "", wantErr: true},
		{name: "dot-dot traversal", value: "../../admin", wantErr: true},
		{name: "forward slash", value: "path/to/thing", wantErr: true},
		{name: "backslash", value: "path\\to\\thing", wantErr: true},
		{name: "embedded dot-dot", value: "ok..notok", wantErr: true},
		{name: "question mark", value: "id?param=val", wantErr: true},
		{name: "hash fragment", value: "id#frag", wantErr: true},
		{name: "percent encoding", value: "id%2F", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateGraphID("test_field", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGraphID(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

func TestValidateValuesGrid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  [][]any
		wantErr bool
	}{
		{name: "empty", values: [][]any{}, wantErr: false},
		{name: "single row", values: [][]any{{"a", "b"}}, wantErr: false},
		{name: "consistent columns", values: [][]any{{"a", "b"}, {"c", "d"}}, wantErr: false},
		{name: "inconsistent columns", values: [][]any{{"a", "b", "c"}, {"d", "e"}}, wantErr: true},
		{name: "second row longer", values: [][]any{{"a"}, {"b", "c"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateValuesGrid(tt.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateValuesGrid() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}
