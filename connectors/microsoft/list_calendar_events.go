package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listCalendarEventsAction implements connectors.Action for microsoft.list_calendar_events.
// It lists upcoming events via GET /me/events using the Microsoft Graph API.
type listCalendarEventsAction struct {
	conn *MicrosoftConnector
}

// listCalendarEventsParams is the user-facing parameter schema.
type listCalendarEventsParams struct {
	CalendarID string `json:"calendar_id,omitempty"`
	Top        int    `json:"top"`
}

func (p *listCalendarEventsParams) defaults() {
	if p.Top <= 0 {
		p.Top = 10
	}
	if p.Top > 50 {
		p.Top = 50
	}
}

// graphEventsResponse is the Microsoft Graph API response for listing events.
type graphEventsResponse struct {
	Value []graphEventItem `json:"value"`
}

type graphEventItem struct {
	ID        string              `json:"id"`
	Subject   string              `json:"subject"`
	Start     graphDateTimeZone   `json:"start"`
	End       graphDateTimeZone   `json:"end"`
	Location  *graphEventLocation `json:"location,omitempty"`
	Organizer *graphMailAddress   `json:"organizer,omitempty"`
	WebLink   string              `json:"webLink"`
	IsAllDay  bool                `json:"isAllDay"`
}

// eventSummary is the simplified response returned to the caller.
type eventSummary struct {
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	Start     string `json:"start"`
	End       string `json:"end"`
	TimeZone  string `json:"time_zone"`
	Location  string `json:"location"`
	Organizer string `json:"organizer"`
	WebLink   string `json:"web_link"`
	IsAllDay  bool   `json:"is_all_day"`
}

// Execute lists upcoming calendar events.
func (a *listCalendarEventsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listCalendarEventsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.defaults()

	var path string
	if params.CalendarID == "" {
		path = fmt.Sprintf("/me/events?$top=%d&$orderby=start/dateTime&$select=id,subject,start,end,location,organizer,webLink,isAllDay", params.Top)
	} else {
		path = fmt.Sprintf("/me/calendars/%s/events?$top=%d&$orderby=start/dateTime&$select=id,subject,start,end,location,organizer,webLink,isAllDay",
			url.PathEscape(params.CalendarID), params.Top)
	}

	var resp graphEventsResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]eventSummary, len(resp.Value))
	for i, ev := range resp.Value {
		var location string
		if ev.Location != nil {
			location = ev.Location.DisplayName
		}
		var organizer string
		if ev.Organizer != nil {
			organizer = ev.Organizer.EmailAddress.Address
		}
		summaries[i] = eventSummary{
			ID:        ev.ID,
			Subject:   ev.Subject,
			Start:     ev.Start.DateTime,
			End:       ev.End.DateTime,
			TimeZone:  ev.Start.TimeZone,
			Location:  location,
			Organizer: organizer,
			WebLink:   ev.WebLink,
			IsAllDay:  ev.IsAllDay,
		}
	}

	return connectors.JSONResult(summaries)
}
