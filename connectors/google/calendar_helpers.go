package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const maxCalendarIDLength = 1024

// calendarLookup is the Calendar API result for a calendar_id (including
// "primary" and omitted, which default to the user's primary calendar).
type calendarLookup struct {
	ID      string
	Summary string
}

// isValidCalendarID rejects values that would break out of the Calendar API
// path even after URL-escaping (slashes, query/fragment, control chars).
func isValidCalendarID(s string) bool {
	if s == "" || len(s) > maxCalendarIDLength {
		return false
	}
	if strings.ContainsAny(s, "/?#\n\r\x00") {
		return false
	}
	return true
}

func normalizeCalendarID(calendarID string) string {
	if calendarID == "" {
		return "primary"
	}
	return calendarID
}

// fetchCalendar loads id+summary via GET /calendars/{calendarId}.
// Empty calendarID is treated as "primary".
func (c *GoogleConnector) fetchCalendar(ctx context.Context, creds connectors.Credentials, calendarID string) (calendarLookup, error) {
	calendarID = normalizeCalendarID(calendarID)
	if !isValidCalendarID(calendarID) {
		return calendarLookup{}, fmt.Errorf("invalid calendar_id")
	}

	var resp struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	}
	getURL := c.calendarBaseURL + "/calendars/" + url.PathEscape(calendarID) + "?fields=id,summary"
	if err := c.doJSON(ctx, creds, http.MethodGet, getURL, nil, &resp); err != nil {
		return calendarLookup{}, err
	}
	return calendarLookup{ID: resp.ID, Summary: resp.Summary}, nil
}

// lookupCalendar is fetchCalendar plus a fail-closed check that the Calendar
// API returned a canonical id (used for $meta.calendar_id matching).
func (c *GoogleConnector) lookupCalendar(ctx context.Context, creds connectors.Credentials, calendarID string) (calendarLookup, error) {
	cal, err := c.fetchCalendar(ctx, creds, calendarID)
	if err != nil {
		return calendarLookup{}, err
	}
	if cal.ID == "" {
		return calendarLookup{}, fmt.Errorf("calendar %q has no id", normalizeCalendarID(calendarID))
	}
	return cal, nil
}

// normalizeCalendarTimeParams rewrites common time-parameter aliases in the raw
// JSON so the typed unmarshal succeeds even when an LLM agent sends "start"/"end"
// instead of the schema-defined "start_time"/"end_time". Only absent canonical
// keys are backfilled — if the agent sends both "start" and "start_time", the
// canonical key wins and the alias is ignored.
func normalizeCalendarTimeParams(raw json.RawMessage) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw // let the caller's Unmarshal report the error
	}

	aliases := map[string]string{
		"start": "start_time",
		"end":   "end_time",
	}

	changed := false
	for alias, canonical := range aliases {
		if _, hasCanonical := m[canonical]; hasCanonical {
			if _, hasAlias := m[alias]; hasAlias {
				delete(m, alias)
				changed = true
			}
			continue
		}
		if val, hasAlias := m[alias]; hasAlias {
			m[canonical] = val
			delete(m, alias)
			changed = true
		}
	}

	if !changed {
		return raw
	}

	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// validateTimeRange checks that start and end are valid RFC 3339 timestamps
// and that end is strictly after start. Used by create_calendar_event and
// create_meeting to avoid duplicating time validation logic.
func validateTimeRange(startTime, endTime string) error {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("start_time must be RFC 3339 format: %v", err)}
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("end_time must be RFC 3339 format: %v", err)}
	}
	if !end.After(start) {
		return &connectors.ValidationError{Message: "end_time must be after start_time"}
	}
	return nil
}

// buildAttendees converts a slice of email strings into calendarAttendee
// structs for the Google Calendar API.
func buildAttendees(emails []string) []calendarAttendee {
	if len(emails) == 0 {
		return nil
	}
	attendees := make([]calendarAttendee, len(emails))
	for i, email := range emails {
		attendees[i] = calendarAttendee{Email: email}
	}
	return attendees
}
