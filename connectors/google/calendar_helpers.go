package google

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

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
