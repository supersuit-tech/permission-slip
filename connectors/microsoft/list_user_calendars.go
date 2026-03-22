package microsoft

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	msCalendarsPageSize   = 200
	msCalendarsMaxPages   = 4
	msCalendarsSelectCols = "id,name,isDefaultCalendar,owner"
)

type graphCalendarListResponse struct {
	ODataNextLink string `json:"@odata.nextLink"`
	Value         []struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		IsDefaultCalendar bool   `json:"isDefaultCalendar"`
		Owner             *struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"owner,omitempty"`
	} `json:"value"`
}

// ListUserCalendars returns calendars from Microsoft Graph GET /me/calendars,
// following @odata.nextLink until exhausted or msCalendarsMaxPages.
func (c *MicrosoftConnector) ListUserCalendars(ctx context.Context, creds connectors.Credentials) ([]connectors.UserCalendar, error) {
	q := url.Values{}
	q.Set("$top", fmt.Sprintf("%d", msCalendarsPageSize))
	q.Set("$select", msCalendarsSelectCols)
	nextPath := "/me/calendars?" + q.Encode()

	var out []connectors.UserCalendar
	for page := 0; page < msCalendarsMaxPages; page++ {
		var raw graphCalendarListResponse
		if err := c.doRequest(ctx, http.MethodGet, nextPath, creds, nil, &raw); err != nil {
			return nil, err
		}
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
		if raw.ODataNextLink == "" {
			break
		}
		var err error
		nextPath, err = graphRelativePathFromNextLink(c.baseURL, raw.ODataNextLink)
		if err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("invalid Microsoft Graph nextLink: %v", err)}
		}
	}
	return out, nil
}

// graphRelativePathFromNextLink converts a full nextLink URL into a path+query
// string relative to baseURL (what doRequest expects).
func graphRelativePathFromNextLink(baseURL, nextLink string) (string, error) {
	base, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil {
		return "", err
	}
	n, err := url.Parse(nextLink)
	if err != nil {
		return "", err
	}
	if n.Scheme != base.Scheme || n.Host != base.Host {
		return "", fmt.Errorf("nextLink host/scheme does not match connector baseURL")
	}
	rel := strings.TrimPrefix(n.Path, base.Path)
	if rel == "" {
		rel = "/"
	} else if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	if n.RawQuery != "" {
		rel += "?" + n.RawQuery
	}
	return rel, nil
}
