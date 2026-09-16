package google

import (
	"strings"
	"testing"
	"time"
)

func TestValidateRecurrence(t *testing.T) {
	t.Parallel()

	if err := validateRecurrence(nil); err == nil {
		t.Fatal("expected error for empty recurrence")
	}
	if err := validateRecurrence([]string{"EXDATE:20260120"}); err == nil {
		t.Fatal("expected error when RRULE is missing")
	}
	if err := validateRecurrence([]string{"FREQ=WEEKLY"}); err == nil {
		t.Fatal("expected error for bare FREQ without RRULE: prefix")
	}
	if err := validateRecurrence([]string{"RRULE:FREQ=WEEKLY;BYDAY=TU"}); err != nil {
		t.Fatalf("valid RRULE rejected: %v", err)
	}
	if err := validateRecurrence([]string{"RRULE:FREQ=WEEKLY", "EXDATE:20260120T150000Z"}); err != nil {
		t.Fatalf("RRULE+EXDATE rejected: %v", err)
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
