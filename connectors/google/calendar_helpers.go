package google

import (
	"fmt"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

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
