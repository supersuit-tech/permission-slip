package microsoft

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ListCalendars returns the user's calendars (Microsoft Graph GET /me/calendars).
// Implements connectors.CalendarLister.
func (c *MicrosoftConnector) ListCalendars(ctx context.Context, creds connectors.Credentials) ([]connectors.CalendarListItem, error) {
	if err := c.ValidateCredentials(ctx, creds); err != nil {
		return nil, err
	}

	var out []connectors.CalendarListItem
	nextPath := "/me/calendars?$select=id,name,isDefaultCalendar&$top=100"
	for i := 0; i < 20 && nextPath != ""; i++ {
		var payload struct {
			Value []struct {
				ID                string `json:"id"`
				Name              string `json:"name"`
				IsDefaultCalendar bool   `json:"isDefaultCalendar"`
			} `json:"value"`
			NextLink string `json:"@odata.nextLink"`
		}
		if err := c.doRequest(ctx, http.MethodGet, nextPath, creds, nil, &payload); err != nil {
			return nil, err
		}
		for _, it := range payload.Value {
			label := it.Name
			if label == "" {
				label = it.ID
			}
			out = append(out, connectors.CalendarListItem{
				ID:                it.ID,
				Name:              label,
				IsDefaultCalendar: it.IsDefaultCalendar,
			})
		}
		nextPath = graphNextRelativePath(c.baseURL, payload.NextLink)
	}
	return out, nil
}

// CalendarListCredentialActionType implements connectors.CalendarLister.
func (c *MicrosoftConnector) CalendarListCredentialActionType() string {
	return "microsoft.list_calendar_events"
}

// graphNextRelativePath turns an absolute @odata.nextLink into a path+query
// relative to the connector base URL, or empty if parsing fails.
func graphNextRelativePath(baseURL, nextLink string) string {
	if nextLink == "" {
		return ""
	}
	u, err := url.Parse(nextLink)
	if err != nil {
		return ""
	}
	base := strings.TrimSuffix(baseURL, "/")
	full := strings.TrimSuffix(u.String(), "/")
	if !strings.HasPrefix(full, base+"/") && !strings.HasPrefix(full, base+"?") {
		return ""
	}
	rel := full[len(base):]
	if rel == "" || rel == "/" {
		return ""
	}
	if rel[0] == '/' || rel[0] == '?' {
		return rel
	}
	return "/" + rel
}
