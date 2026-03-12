package google

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCalendarTimeParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantKey  string
		wantVal  string
		wantKey2 string
		wantVal2 string
	}{
		{
			name:     "aliases start/end to start_time/end_time",
			input:    `{"summary":"Hello","start":"2026-03-13T10:00:00-05:00","end":"2026-03-13T11:00:00-05:00"}`,
			wantKey:  "start_time",
			wantVal:  "2026-03-13T10:00:00-05:00",
			wantKey2: "end_time",
			wantVal2: "2026-03-13T11:00:00-05:00",
		},
		{
			name:     "canonical names preserved when present",
			input:    `{"summary":"Hello","start_time":"2026-03-13T10:00:00-05:00","end_time":"2026-03-13T11:00:00-05:00"}`,
			wantKey:  "start_time",
			wantVal:  "2026-03-13T10:00:00-05:00",
			wantKey2: "end_time",
			wantVal2: "2026-03-13T11:00:00-05:00",
		},
		{
			name:     "canonical wins over alias when both present",
			input:    `{"start_time":"canonical","start":"alias","end_time":"canonical2","end":"alias2"}`,
			wantKey:  "start_time",
			wantVal:  "canonical",
			wantKey2: "end_time",
			wantVal2: "canonical2",
		},
		{
			name:     "partial alias only start",
			input:    `{"start":"2026-03-13T10:00:00-05:00","end_time":"2026-03-13T11:00:00-05:00"}`,
			wantKey:  "start_time",
			wantVal:  "2026-03-13T10:00:00-05:00",
			wantKey2: "end_time",
			wantVal2: "2026-03-13T11:00:00-05:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := normalizeCalendarTimeParams(json.RawMessage(tt.input))

			var m map[string]json.RawMessage
			if err := json.Unmarshal(result, &m); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			checkKey := func(key, want string) {
				raw, ok := m[key]
				if !ok {
					t.Errorf("expected key %q in result", key)
					return
				}
				var got string
				if err := json.Unmarshal(raw, &got); err != nil {
					t.Errorf("failed to unmarshal %q: %v", key, err)
					return
				}
				if got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}

			checkKey(tt.wantKey, tt.wantVal)
			checkKey(tt.wantKey2, tt.wantVal2)
		})
	}
}

func TestNormalizeCalendarTimeParams_AliasRemoved(t *testing.T) {
	t.Parallel()
	input := `{"summary":"Hello","start":"2026-03-13T10:00:00-05:00","end":"2026-03-13T11:00:00-05:00"}`
	result := normalizeCalendarTimeParams(json.RawMessage(input))

	var m map[string]json.RawMessage
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := m["start"]; ok {
		t.Error("alias key 'start' should have been removed")
	}
	if _, ok := m["end"]; ok {
		t.Error("alias key 'end' should have been removed")
	}
}

func TestNormalizeCalendarTimeParams_AliasRemovedWhenCanonicalPresent(t *testing.T) {
	t.Parallel()
	input := `{"start_time":"canonical","start":"alias","end_time":"canonical2","end":"alias2"}`
	result := normalizeCalendarTimeParams(json.RawMessage(input))

	var m map[string]json.RawMessage
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if _, ok := m["start"]; ok {
		t.Error("alias key 'start' should have been removed even when canonical is present")
	}
	if _, ok := m["end"]; ok {
		t.Error("alias key 'end' should have been removed even when canonical is present")
	}
}

func TestNormalizeCalendarTimeParams_InvalidJSON(t *testing.T) {
	t.Parallel()
	input := json.RawMessage(`not valid json`)
	result := normalizeCalendarTimeParams(input)
	// Should return input unchanged
	if string(result) != string(input) {
		t.Errorf("expected unchanged input for invalid JSON, got %s", string(result))
	}
}
