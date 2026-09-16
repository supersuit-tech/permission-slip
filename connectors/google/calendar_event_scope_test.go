package google

import (
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestParseCalendarEventScope(t *testing.T) {
	t.Parallel()

	for _, scope := range []string{"", "instance", "series", "this_and_following"} {
		if _, err := parseCalendarEventScope(scope); err != nil {
			t.Errorf("scope %q should be valid: %v", scope, err)
		}
	}
	if _, err := parseCalendarEventScope("THISANDFUTURE"); err == nil {
		t.Fatal("expected error for unknown scope")
	}
}

func TestValidateScopeAgainstEventID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		scope         string
		eventID       string
		instanceStart string
		wantErr       string
	}{
		{name: "omitted scope passthrough", eventID: "abc123"},
		{name: "instance with instance id", scope: "instance", eventID: "abc123_20260120T150000Z"},
		{name: "instance with master plus start", scope: "instance", eventID: "abc123", instanceStart: "2026-01-20T15:00:00Z"},
		{name: "series with master", scope: "series", eventID: "abc123"},
		{name: "this_and_following with instance id", scope: "this_and_following", eventID: "abc123_20260120T150000Z"},
		{
			name:    "instance with only master",
			scope:   "instance",
			eventID: "abc123",
			wantErr: "expanded instance event_id",
		},
		{
			name:    "series with instance id",
			scope:   "series",
			eventID: "abc123_20260120T150000Z",
			wantErr: "series master event_id",
		},
		{
			name:    "this_and_following with only master",
			scope:   "this_and_following",
			eventID: "abc123",
			wantErr: "this_and_following requires",
		},
		{
			name:          "instance_start without scope",
			eventID:       "abc123",
			instanceStart: "2026-01-20T15:00:00Z",
			wantErr:       "instance_start requires scope",
		},
		{
			name:          "instance_start with series",
			scope:         "series",
			eventID:       "abc123",
			instanceStart: "2026-01-20T15:00:00Z",
			wantErr:       "instance_start cannot be used with scope=series",
		},
		{
			name:          "bad instance_start",
			scope:         "instance",
			eventID:       "abc123",
			instanceStart: "next Tuesday",
			wantErr:       "RFC 3339",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateScopeAgainstEventID(tt.scope, tt.eventID, tt.instanceStart)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestClassifyCalendarEvent(t *testing.T) {
	t.Parallel()

	if got := classifyCalendarEvent(&calendarEventResource{ID: "a"}); got != calendarEventKindSingle {
		t.Errorf("single: got %s", got)
	}
	if got := classifyCalendarEvent(&calendarEventResource{ID: "a", Recurrence: []string{"RRULE:FREQ=WEEKLY"}}); got != calendarEventKindSeriesMaster {
		t.Errorf("master: got %s", got)
	}
	if got := classifyCalendarEvent(&calendarEventResource{ID: "a_20260120T150000Z", RecurringEventID: "a"}); got != calendarEventKindInstance {
		t.Errorf("instance: got %s", got)
	}
}

func TestIsFirstSeriesInstance(t *testing.T) {
	t.Parallel()

	master := &calendarEventResource{
		Start: calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
	}
	first := &calendarEventResource{
		OriginalStart: calendarEventDateTime{DateTime: "2026-01-06T15:00:00Z"},
	}
	later := &calendarEventResource{
		OriginalStart: calendarEventDateTime{DateTime: "2026-01-13T15:00:00Z"},
	}
	if !isFirstSeriesInstance(master, first) {
		t.Error("first occurrence should match master start")
	}
	if isFirstSeriesInstance(master, later) {
		t.Error("later occurrence should not be first")
	}
}

func TestInstanceStartMatches(t *testing.T) {
	t.Parallel()

	ev := &calendarEventResource{
		OriginalStart: calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
		Start:         calendarEventDateTime{DateTime: "2026-01-20T16:00:00-05:00"},
	}
	start, _ := time.Parse(time.RFC3339, "2026-01-20T15:00:00Z")
	if !instanceStartMatches(ev, start, false) {
		t.Error("should match originalStartTime in UTC")
	}
	miss, _ := time.Parse(time.RFC3339, "2026-01-27T15:00:00Z")
	if instanceStartMatches(ev, miss, false) {
		t.Error("should not match a different week")
	}

	allDay := &calendarEventResource{
		OriginalStart: calendarEventDateTime{Date: "2026-01-20"},
	}
	day, _ := time.Parse("2006-01-02", "2026-01-20")
	if !instanceStartMatches(allDay, day, true) {
		t.Error("should match all-day original start")
	}
}

func TestNewSeriesBodyFromMaster_KeepsRewrittenRecurrence(t *testing.T) {
	t.Parallel()

	master := &calendarEventResource{
		Summary:    "Standup",
		Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=20"},
		Attendees:  []calendarAttendee{{Email: "a@example.com"}},
	}
	instance := &calendarEventResource{
		Start: calendarEventDateTime{DateTime: "2026-01-20T15:00:00Z"},
		End:   calendarEventDateTime{DateTime: "2026-01-20T16:00:00Z"},
	}
	updates := map[string]any{
		"summary":    "Moved standup",
		"recurrence": []string{"RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=20"},
	}
	rewritten := []string{"RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=7"}
	body := newSeriesBodyFromMaster(master, instance, updates, rewritten)
	rec, _ := body["recurrence"].([]string)
	if len(rec) != 1 || rec[0] != "RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=7" {
		t.Errorf("updates must not overwrite rewritten recurrence, got %v", body["recurrence"])
	}
	if body["summary"] != "Moved standup" {
		t.Errorf("summary overlay failed: %v", body["summary"])
	}
}
