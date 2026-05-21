package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listCalendarsAction implements connectors.Action for google.list_calendars.
// It lists all calendars accessible to the authenticated user via the Google
// Calendar API GET /users/me/calendarList.
type listCalendarsAction struct {
	conn *GoogleConnector
}

type calendarEntry struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Primary     bool   `json:"primary"`
	Description string `json:"description,omitempty"`
	AccessRole  string `json:"access_role,omitempty"`
	TimeZone    string `json:"time_zone,omitempty"`
}

// Execute returns all calendars visible to the authenticated user.
func (a *listCalendarsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	if len(req.Parameters) > 0 && string(req.Parameters) != "null" {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(req.Parameters, &raw); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	var calendars []calendarEntry
	var pageToken string
	const maxPages = 20
	for page := 0; page < maxPages; page++ {
		q := url.Values{}
		q.Set("maxResults", "250")
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		rawURL := a.conn.calendarBaseURL + "/users/me/calendarList?" + q.Encode()

		var payload struct {
			Items []struct {
				ID          string `json:"id"`
				Summary     string `json:"summary"`
				SummaryO    string `json:"summaryOverride"`
				Description string `json:"description"`
				AccessRole  string `json:"accessRole"`
				TimeZone    string `json:"timeZone"`
				Primary     *bool  `json:"primary"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, rawURL, nil, &payload); err != nil {
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
			calendars = append(calendars, calendarEntry{
				ID:          it.ID,
				Summary:     label,
				Primary:     primary,
				Description: it.Description,
				AccessRole:  it.AccessRole,
				TimeZone:    it.TimeZone,
			})
		}
		if payload.NextPageToken == "" {
			break
		}
		pageToken = payload.NextPageToken
	}

	if calendars == nil {
		calendars = []calendarEntry{}
	}

	return connectors.JSONResult(map[string]any{
		"calendars": calendars,
	})
}
