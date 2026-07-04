package db

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateDataWindowConstraintShape(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "last_days only", raw: `{"last_days":30}`},
		{name: "starts_at only", raw: `{"starts_at":"2026-07-01T00:00:00Z"}`},
		{name: "range", raw: `{"starts_at":"2026-07-01T00:00:00Z","ends_at":"2026-08-01T00:00:00Z"}`},
		{name: "last_days with ends_at", raw: `{"last_days":7,"ends_at":"2026-08-01T00:00:00Z"}`},
		{name: "empty object", raw: `{}`, wantErr: true},
		{name: "last_days and starts_at", raw: `{"last_days":30,"starts_at":"2026-07-01T00:00:00Z"}`, wantErr: true},
		{name: "ends before starts", raw: `{"starts_at":"2026-08-01T00:00:00Z","ends_at":"2026-07-01T00:00:00Z"}`, wantErr: true},
		{name: "last_days zero", raw: `{"last_days":0}`, wantErr: true},
		{name: "unknown field", raw: `{"last_days":30,"foo":"bar"}`, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDataWindowConstraintShape(json.RawMessage(tc.raw))
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolveDataWindowConstraint_LastDays(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	window, err := ResolveDataWindowConstraint(json.RawMessage(`{"last_days":30}`), now)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if window.StartsAt == nil {
		t.Fatal("expected starts_at")
	}
	want := now.AddDate(0, 0, -30)
	if !window.StartsAt.Equal(want) {
		t.Fatalf("starts_at = %v, want %v", window.StartsAt, want)
	}
}

func TestApplyDataWindowToParams_InjectWhenAbsent(t *testing.T) {
	t.Parallel()
	floor := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	window := ResolvedDataWindow{StartsAt: &floor}
	out, err := ApplyDataWindowToParams(
		json.RawMessage(`{"chat_id":42}`),
		window,
		"start", "end",
	)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["start"] != floor.Format(time.RFC3339) {
		t.Fatalf("start = %q, want %q", m["start"], floor.Format(time.RFC3339))
	}
}

func TestApplyDataWindowToParams_ClampOlderStart(t *testing.T) {
	t.Parallel()
	floor := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	window := ResolvedDataWindow{StartsAt: &floor}
	out, err := ApplyDataWindowToParams(
		json.RawMessage(`{"chat_id":42,"start":"2020-01-01T00:00:00Z"}`),
		window,
		"start", "end",
	)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]string
	_ = json.Unmarshal(out, &m)
	if m["start"] != floor.Format(time.RFC3339) {
		t.Fatalf("start = %q, want clamped floor", m["start"])
	}
}

func TestApplyDataWindowToParams_PassThroughNarrowerStart(t *testing.T) {
	t.Parallel()
	floor := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	agentStart := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	window := ResolvedDataWindow{StartsAt: &floor}
	out, err := ApplyDataWindowToParams(
		json.RawMessage(`{"chat_id":42,"start":"2026-07-01T00:00:00Z"}`),
		window,
		"start", "end",
	)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	var m map[string]string
	_ = json.Unmarshal(out, &m)
	if m["start"] != agentStart.Format(time.RFC3339) {
		t.Fatalf("start = %q, want agent value", m["start"])
	}
}

func TestValidateParametersAgainstConfig_IgnoresDataWindow(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"chat_id":42,"$data_window":{"last_days":30}}`)
	exec := json.RawMessage(`{"chat_id":42}`)
	if err := ValidateParametersAgainstConfig(config, exec, nil); err != nil {
		t.Fatalf("expected match, got %v", err)
	}
}

func TestValidateParametersAgainstConfig_RejectsExtraExecKeyWithDataWindow(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"chat_id":42,"$data_window":{"last_days":30}}`)
	exec := json.RawMessage(`{"chat_id":42,"bogus":"x"}`)
	if err := ValidateParametersAgainstConfig(config, exec, nil); err == nil {
		t.Fatal("expected extra key rejection")
	}
}
