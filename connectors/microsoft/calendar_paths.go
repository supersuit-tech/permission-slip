package microsoft

import (
	"fmt"
	"net/url"
	"strings"
)

// microsoftCalendarEventsBasePath returns the Graph resource path for a
// calendar's events collection. Empty calendarID uses the signed-in user's
// default calendar (/me/events).
func microsoftCalendarEventsBasePath(calendarID string) string {
	calendarID = strings.TrimSpace(calendarID)
	if calendarID == "" {
		return "/me/events"
	}
	enc := url.PathEscape(calendarID)
	return fmt.Sprintf("/me/calendars/%s/events", enc)
}
