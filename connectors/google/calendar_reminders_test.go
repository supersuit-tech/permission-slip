package google

import (
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func intPtr(v int) *int { return &v }

func boolPtr(v bool) *bool { return &v }

func TestResolveEventReminders_Omitted(t *testing.T) {
	t.Parallel()
	got, err := resolveEventReminders(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil payload when reminders are omitted, got %+v", got)
	}
}

func TestResolveEventReminders_ShorthandPopups(t *testing.T) {
	t.Parallel()
	got, err := resolveEventReminders(nil, []int{5, 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected payload")
	}
	if got.UseDefault {
		t.Error("expected useDefault false")
	}
	if len(got.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(got.Overrides))
	}
	if got.Overrides[0].Method != "popup" || got.Overrides[0].Minutes != 5 {
		t.Errorf("override[0] = %+v, want popup/5", got.Overrides[0])
	}
	if got.Overrides[1].Method != "popup" || got.Overrides[1].Minutes != 1 {
		t.Errorf("override[1] = %+v, want popup/1", got.Overrides[1])
	}
}

func TestResolveEventReminders_CustomOverrides(t *testing.T) {
	t.Parallel()
	got, err := resolveEventReminders(&eventRemindersParams{
		UseDefault: boolPtr(false),
		Overrides: []eventReminderOverride{
			{Method: "popup", Minutes: intPtr(5)},
			{Method: "popup", Minutes: intPtr(1)},
			{Method: "email", Minutes: intPtr(1440)},
		},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UseDefault || len(got.Overrides) != 3 {
		t.Fatalf("got %+v", got)
	}
	if got.Overrides[2].Method != "email" || got.Overrides[2].Minutes != 1440 {
		t.Errorf("email override = %+v", got.Overrides[2])
	}
}

func TestResolveEventReminders_UseDefault(t *testing.T) {
	t.Parallel()
	got, err := resolveEventReminders(&eventRemindersParams{UseDefault: boolPtr(true)}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.UseDefault {
		t.Error("expected useDefault true")
	}
	if len(got.Overrides) != 0 {
		t.Errorf("expected no overrides, got %+v", got.Overrides)
	}
}

func TestResolveEventReminders_NoReminders(t *testing.T) {
	t.Parallel()
	got, err := resolveEventReminders(&eventRemindersParams{UseDefault: boolPtr(false)}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UseDefault || len(got.Overrides) != 0 {
		t.Errorf("expected disabled reminders, got %+v", got)
	}
}

func TestResolveEventReminders_ValidationErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		params   *eventRemindersParams
		minutes  []int
		contains string
	}{
		{
			name:     "both fields",
			params:   &eventRemindersParams{UseDefault: boolPtr(true)},
			minutes:  []int{5},
			contains: "mutually exclusive",
		},
		{
			name:     "empty object",
			params:   &eventRemindersParams{},
			contains: "use_default or overrides",
		},
		{
			name: "default with overrides",
			params: &eventRemindersParams{
				UseDefault: boolPtr(true),
				Overrides:  []eventReminderOverride{{Method: "popup", Minutes: intPtr(5)}},
			},
			contains: "use_default is true",
		},
		{
			name: "sms method",
			params: &eventRemindersParams{
				Overrides: []eventReminderOverride{{Method: "sms", Minutes: intPtr(5)}},
			},
			contains: "popup or email",
		},
		{
			name: "missing minutes",
			params: &eventRemindersParams{
				Overrides: []eventReminderOverride{{Method: "popup"}},
			},
			contains: "minutes is required",
		},
		{
			name:     "negative shorthand",
			minutes:  []int{-1},
			contains: "non-negative",
		},
		{
			name:     "too many shorthand",
			minutes:  []int{1, 2, 3, 4, 5, 6},
			contains: "at most 5",
		},
		{
			name: "minutes too large",
			params: &eventRemindersParams{
				Overrides: []eventReminderOverride{{Method: "Email", Minutes: intPtr(40321)}},
			},
			contains: "at most 40320",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := resolveEventReminders(tc.params, tc.minutes)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
			if !strings.Contains(err.Error(), tc.contains) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.contains)
			}
		})
	}
}
