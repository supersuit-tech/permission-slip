package google

import (
	"context"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ListCalendars returns calendars visible to the user (Google Calendar API
// calendarList). Implements connectors.CalendarLister.
func (c *GoogleConnector) ListCalendars(ctx context.Context, creds connectors.Credentials) ([]connectors.CalendarListItem, error) {
	if err := c.ValidateCredentials(ctx, creds); err != nil {
		return nil, err
	}

	const pageSize = 250
	const maxPages = 20
	var out []connectors.CalendarListItem
	var pageToken string
	for page := 0; page < maxPages; page++ {
		q := url.Values{}
		q.Set("maxResults", fmt.Sprintf("%d", pageSize))
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		rawURL := c.calendarBaseURL + "/users/me/calendarList?" + q.Encode()

		var payload struct {
			Items         []struct {
				ID       string `json:"id"`
				Summary  string `json:"summary"`
				Primary  *bool  `json:"primary"`
				SummaryO string `json:"summaryOverride"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := c.doJSON(ctx, creds, "GET", rawURL, nil, &payload); err != nil {
			return nil, err
		}
		for _, it := range payload.Items {
			label := it.Summary
			if label == "" {
				label = it.SummaryO
			}
			if label == "" {
				label = it.ID
			}
			primary := it.Primary != nil && *it.Primary
			out = append(out, connectors.CalendarListItem{
				ID:      it.ID,
				Summary: label,
				Primary: primary,
			})
		}
		if payload.NextPageToken == "" {
			return out, nil
		}
		pageToken = payload.NextPageToken
	}
	return out, nil
}

// CalendarListCredentialActionType implements connectors.CalendarLister.
func (c *GoogleConnector) CalendarListCredentialActionType() string {
	return "google.list_calendar_events"
}
