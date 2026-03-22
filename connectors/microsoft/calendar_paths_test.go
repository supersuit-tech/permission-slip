package microsoft

import "testing"

func TestMicrosoftCalendarEventsBasePath_Default(t *testing.T) {
	t.Parallel()
	if got := microsoftCalendarEventsBasePath(""); got != "/me/events" {
		t.Errorf("empty id: got %q", got)
	}
	if got := microsoftCalendarEventsBasePath("   "); got != "/me/events" {
		t.Errorf("whitespace: got %q", got)
	}
}

func TestMicrosoftCalendarEventsBasePath_Escaped(t *testing.T) {
	t.Parallel()
	got := microsoftCalendarEventsBasePath("cal/with/slash")
	if got != "/me/calendars/cal%2Fwith%2Fslash/events" {
		t.Errorf("got %q", got)
	}
}
