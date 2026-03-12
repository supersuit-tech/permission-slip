package connectors

import (
	"encoding/json"
	"testing"
)

func TestNormalizeParameters(t *testing.T) {
	aliases := map[string]string{
		"start": "start_time",
		"end":   "end_time",
	}

	tests := []struct {
		name    string
		aliases map[string]string
		params  string
		want    string
	}{
		{
			name:    "alias rewritten to canonical",
			aliases: aliases,
			params:  `{"start":"2024-01-15T09:00:00Z","end":"2024-01-15T10:00:00Z","summary":"Meeting"}`,
			want:    `{"start_time":"2024-01-15T09:00:00Z","end_time":"2024-01-15T10:00:00Z","summary":"Meeting"}`,
		},
		{
			name:    "canonical names pass through unchanged",
			aliases: aliases,
			params:  `{"start_time":"2024-01-15T09:00:00Z","end_time":"2024-01-15T10:00:00Z","summary":"Meeting"}`,
			want:    `{"start_time":"2024-01-15T09:00:00Z","end_time":"2024-01-15T10:00:00Z","summary":"Meeting"}`,
		},
		{
			name:    "canonical wins when both present",
			aliases: aliases,
			params:  `{"start":"alias-val","start_time":"canonical-val","summary":"Meeting"}`,
			want:    `{"start_time":"canonical-val","summary":"Meeting"}`,
		},
		{
			name:    "partial alias only start",
			aliases: aliases,
			params:  `{"start":"2024-01-15T09:00:00Z","summary":"Meeting"}`,
			want:    `{"start_time":"2024-01-15T09:00:00Z","summary":"Meeting"}`,
		},
		{
			name:    "no matching keys passes through",
			aliases: aliases,
			params:  `{"summary":"Meeting","location":"Office"}`,
			want:    `{"summary":"Meeting","location":"Office"}`,
		},
		{
			name:    "nil aliases map returns unchanged",
			aliases: nil,
			params:  `{"start":"2024-01-15T09:00:00Z"}`,
			want:    `{"start":"2024-01-15T09:00:00Z"}`,
		},
		{
			name:    "empty aliases map returns unchanged",
			aliases: map[string]string{},
			params:  `{"start":"2024-01-15T09:00:00Z"}`,
			want:    `{"start":"2024-01-15T09:00:00Z"}`,
		},
		{
			name:    "invalid JSON returns original unchanged",
			aliases: aliases,
			params:  `not-json`,
			want:    `not-json`,
		},
		{
			name:    "empty params returns unchanged",
			aliases: aliases,
			params:  ``,
			want:    ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeParameters(tt.aliases, json.RawMessage(tt.params))
			if !jsonEqual(t, string(got), tt.want) {
				t.Errorf("NormalizeParameters() = %s, want %s", got, tt.want)
			}
		})
	}
}

// jsonEqual compares two JSON strings by unmarshaling both and comparing
// the resulting maps, so key ordering doesn't matter.
func jsonEqual(t *testing.T, a, b string) bool {
	t.Helper()
	if a == b {
		return true
	}
	var ma, mb map[string]json.RawMessage
	if err := json.Unmarshal([]byte(a), &ma); err != nil {
		return a == b
	}
	if err := json.Unmarshal([]byte(b), &mb); err != nil {
		return a == b
	}
	if len(ma) != len(mb) {
		return false
	}
	for k, va := range ma {
		vb, ok := mb[k]
		if !ok {
			return false
		}
		if string(va) != string(vb) {
			return false
		}
	}
	return true
}
