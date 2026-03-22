package microsoft

import (
	"context"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type graphCalendarListResponse struct {
	Value []struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		IsDefaultCalendar bool   `json:"isDefaultCalendar"`
		Owner             *struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"owner,omitempty"`
	} `json:"value"`
}

// ListUserCalendars returns calendars from Microsoft Graph GET /me/calendars.
func (c *MicrosoftConnector) ListUserCalendars(ctx context.Context, creds connectors.Credentials) ([]connectors.UserCalendar, error) {
	q := url.Values{}
	q.Set("$top", "50")
	q.Set("$select", "id,name,isDefaultCalendar,owner")
	path := "/me/calendars?" + q.Encode()

	var raw graphCalendarListResponse
	if err := c.doRequest(ctx, http.MethodGet, path, creds, nil, &raw); err != nil {
		return nil, err
	}

	out := make([]connectors.UserCalendar, 0, len(raw.Value))
	for _, it := range raw.Value {
		if it.ID == "" {
			continue
		}
		name := it.Name
		if name == "" {
			name = it.ID
		}
		var desc string
		if it.Owner != nil && it.Owner.Address != "" {
			desc = it.Owner.Address
		}
		out = append(out, connectors.UserCalendar{
			ID:          it.ID,
			Name:        name,
			Description: desc,
			IsPrimary:   it.IsDefaultCalendar,
		})
	}
	return out, nil
}
