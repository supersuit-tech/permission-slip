package google

import (
	"context"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// googleCalendarListResponse is the response shape for calendarList.list.
type googleCalendarListResponse struct {
	Items []struct {
		ID          string `json:"id"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Primary     bool   `json:"primary"`
	} `json:"items"`
}

// ListUserCalendars returns calendars from the Google Calendar API calendarList
// endpoint for the authenticated user.
func (c *GoogleConnector) ListUserCalendars(ctx context.Context, creds connectors.Credentials) ([]connectors.UserCalendar, error) {
	q := url.Values{}
	q.Set("maxResults", "250")
	listURL := c.calendarBaseURL + "/users/me/calendarList?" + q.Encode()

	var raw googleCalendarListResponse
	if err := c.doJSON(ctx, creds, http.MethodGet, listURL, nil, &raw); err != nil {
		return nil, err
	}

	out := make([]connectors.UserCalendar, 0, len(raw.Items))
	for _, it := range raw.Items {
		if it.ID == "" {
			continue
		}
		name := it.Summary
		if name == "" {
			name = it.ID
		}
		out = append(out, connectors.UserCalendar{
			ID:          it.ID,
			Name:        name,
			Description: it.Description,
			IsPrimary:   it.Primary,
		})
	}
	return out, nil
}
