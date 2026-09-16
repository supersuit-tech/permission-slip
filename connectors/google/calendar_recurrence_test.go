package google

import (
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestValidateRecurrence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		lines   []string
		wantErr string
	}{
		{name: "nil", lines: nil, wantErr: "at least one RRULE"},
		{name: "empty", lines: []string{}, wantErr: "at least one RRULE"},
		{name: "weekly rrule", lines: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU"}},
		{name: "lowercase prefix", lines: []string{"rrule:FREQ=WEEKLY;BYDAY=MO"}},
		{name: "rdate with params", lines: []string{"RRULE:FREQ=WEEKLY", "RDATE;VALUE=DATE:20240120"}},
		{name: "exdate", lines: []string{"RRULE:FREQ=WEEKLY", "EXDATE;TZID=America/New_York:20240116T090000"}},
		{name: "rrule plus exdate", lines: []string{"RRULE:FREQ=WEEKLY;BYDAY=TU", "EXDATE:20240116T140000Z"}},
		{name: "exdate only", lines: []string{"EXDATE:20260120"}, wantErr: "at least one RRULE"},
		{name: "bare freq", lines: []string{"FREQ=WEEKLY"}, wantErr: "must be an RRULE"},
		{name: "empty line", lines: []string{"RRULE:FREQ=DAILY", "  "}, wantErr: "must not be empty"},
		{name: "missing colon", lines: []string{"RRULE"}, wantErr: "must be an RRULE"},
		{name: "missing value", lines: []string{"RRULE:"}, wantErr: "must be an RRULE"},
		{name: "dtstart rejected", lines: []string{"DTSTART:20240116T090000Z"}, wantErr: "DTSTART"},
		{name: "dtend rejected", lines: []string{"DTEND:20240116T100000Z"}, wantErr: "DTEND"},
		{name: "unknown property", lines: []string{"SUMMARY:Nope"}, wantErr: "must start with RRULE"},
		{name: "too many lines", lines: makeRecurrenceLines(maxRecurrenceLines + 1), wantErr: "at most"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateRecurrence(tt.lines)
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

func TestValidateRecurrence_LineTooLong(t *testing.T) {
	t.Parallel()
	line := "RRULE:" + strings.Repeat("A", maxRecurrenceLineLen)
	err := validateRecurrence([]string{line})
	if err == nil {
		t.Fatal("expected error for oversized line")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error %q does not mention exceeds", err.Error())
	}
}

func TestEndRecurrenceBefore_Timed(t *testing.T) {
	t.Parallel()

	cutoff, err := time.Parse(time.RFC3339, "2026-01-20T15:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	got, err := endRecurrenceBefore([]string{"RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=20"}, cutoff, false)
	if err != nil {
		t.Fatalf("endRecurrenceBefore: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 line, got %v", got)
	}
	if !strings.Contains(got[0], "UNTIL=20260120T145959Z") {
		t.Errorf("expected UNTIL just before cutoff, got %q", got[0])
	}
	if strings.Contains(got[0], "COUNT=") {
		t.Errorf("COUNT should be removed when adding UNTIL, got %q", got[0])
	}
	if !strings.Contains(got[0], "FREQ=WEEKLY") || !strings.Contains(got[0], "BYDAY=TU") {
		t.Errorf("FREQ/BYDAY should be preserved, got %q", got[0])
	}
}

func TestEndRecurrenceBefore_ReplacesExistingUntil(t *testing.T) {
	t.Parallel()

	cutoff, _ := time.Parse(time.RFC3339, "2026-01-20T15:00:00Z")
	got, err := endRecurrenceBefore([]string{"RRULE:FREQ=WEEKLY;BYDAY=TU;UNTIL=20261231T235959Z"}, cutoff, false)
	if err != nil {
		t.Fatalf("endRecurrenceBefore: %v", err)
	}
	if strings.Contains(got[0], "20261231") {
		t.Errorf("old UNTIL should be replaced, got %q", got[0])
	}
	if !strings.Contains(got[0], "UNTIL=20260120T145959Z") {
		t.Errorf("expected new UNTIL, got %q", got[0])
	}
}

func TestEndRecurrenceBefore_AllDay(t *testing.T) {
	t.Parallel()

	cutoff, _ := time.Parse("2006-01-02", "2026-01-20")
	got, err := endRecurrenceBefore([]string{"RRULE:FREQ=DAILY", "EXDATE:20260115"}, cutoff, true)
	if err != nil {
		t.Fatalf("endRecurrenceBefore: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected RRULE+EXDATE, got %v", got)
	}
	if got[0] != "RRULE:FREQ=DAILY;UNTIL=20260119" {
		t.Errorf("all-day UNTIL should be previous day, got %q", got[0])
	}
	if got[1] != "EXDATE:20260115" {
		t.Errorf("EXDATE should be preserved, got %q", got[1])
	}
}

func TestEndRecurrenceBefore_NoRRULE(t *testing.T) {
	t.Parallel()

	_, err := endRecurrenceBefore([]string{"EXDATE:20260120"}, time.Now(), false)
	if err == nil {
		t.Fatal("expected error when splitting without RRULE")
	}
}

func TestSetRecurrenceCount(t *testing.T) {
	t.Parallel()

	got := setRecurrenceCount([]string{"RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=20"}, 7)
	if got[0] != "RRULE:FREQ=WEEKLY;BYDAY=TU;COUNT=7" {
		t.Errorf("unexpected rewrite: %q", got[0])
	}
}

func TestLooksLikeInstanceEventID(t *testing.T) {
	t.Parallel()

	if !looksLikeInstanceEventID("abc123_20260120T150000Z") {
		t.Error("timed instance id should match")
	}
	if !looksLikeInstanceEventID("abc123_20260120") {
		t.Error("all-day instance id should match")
	}
	if looksLikeInstanceEventID("abc123") {
		t.Error("master id should not match")
	}
	if looksLikeInstanceEventID("abc_notadate") {
		t.Error("non-date suffix should not match")
	}
}

func makeRecurrenceLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "RRULE:FREQ=DAILY"
	}
	return lines
}
