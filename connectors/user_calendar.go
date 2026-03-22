package connectors

import "context"

// UserCalendar is a calendar the signed-in user can access, returned for UI
// dropdowns when configuring action parameters (e.g. calendar_id).
type UserCalendar struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsPrimary   bool   `json:"is_primary"`
}

// CalendarLister is implemented by connectors that can list the user's
// calendars for dashboard configuration UI.
type CalendarLister interface {
	ListUserCalendars(ctx context.Context, creds Credentials) ([]UserCalendar, error)
}
