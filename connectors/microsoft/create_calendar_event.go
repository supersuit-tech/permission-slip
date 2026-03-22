package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCalendarEventAction implements connectors.Action for microsoft.create_calendar_event.
// It creates an event via POST /me/events using the Microsoft Graph API.
type createCalendarEventAction struct {
	conn *MicrosoftConnector
}

// ParameterAliases maps common agent shorthand to the canonical parameter names.
// Agents familiar with Google Calendar may send "start_time"/"end_time" instead
// of the Microsoft convention "start"/"end".
func (a *createCalendarEventAction) ParameterAliases() map[string]string {
	return map[string]string{
		"start_time": "start",
		"end_time":   "end",
	}
}

// createCalendarEventParams is the user-facing parameter schema.
type createCalendarEventParams struct {
	CalendarID string   `json:"calendar_id"`
	Subject    string   `json:"subject"`
	Start      string   `json:"start"`
	End        string   `json:"end"`
	TimeZone   string   `json:"time_zone"`
	Body       string   `json:"body,omitempty"`
	Attendees  []string `json:"attendees,omitempty"`
	Location   string   `json:"location,omitempty"`
}

func (p *createCalendarEventParams) validate() error {
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if p.Start == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start"}
	}
	if p.End == "" {
		return &connectors.ValidationError{Message: "missing required parameter: end"}
	}
	for i, addr := range p.Attendees {
		if err := validateEmail("attendees", i, addr); err != nil {
			return err
		}
	}
	return nil
}

func (p *createCalendarEventParams) defaults() {
	if p.TimeZone == "" {
		p.TimeZone = "UTC"
	}
}

// graphEventRequest is the Microsoft Graph API request body for creating events.
type graphEventRequest struct {
	Subject   string              `json:"subject"`
	Body      *graphEmailBody     `json:"body,omitempty"`
	Start     graphDateTimeZone   `json:"start"`
	End       graphDateTimeZone   `json:"end"`
	Attendees []graphAttendee     `json:"attendees,omitempty"`
	Location  *graphEventLocation `json:"location,omitempty"`
}

// graphEventResponse is the Microsoft Graph API response for creating events.
type graphEventResponse struct {
	ID        string            `json:"id"`
	Subject   string            `json:"subject"`
	Start     graphDateTimeZone `json:"start"`
	End       graphDateTimeZone `json:"end"`
	WebLink   string            `json:"webLink"`
	Organizer *graphMailAddress `json:"organizer,omitempty"`
}

// Execute creates a calendar event via the Microsoft Graph API.
func (a *createCalendarEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCalendarEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.defaults()

	graphReq := graphEventRequest{
		Subject: params.Subject,
		Start: graphDateTimeZone{
			DateTime: params.Start,
			TimeZone: params.TimeZone,
		},
		End: graphDateTimeZone{
			DateTime: params.End,
			TimeZone: params.TimeZone,
		},
	}

	if params.Body != "" {
		graphReq.Body = &graphEmailBody{
			ContentType: "HTML",
			Content:     params.Body,
		}
	}

	if len(params.Attendees) > 0 {
		graphReq.Attendees = make([]graphAttendee, len(params.Attendees))
		for i, addr := range params.Attendees {
			graphReq.Attendees[i].EmailAddress.Address = addr
			graphReq.Attendees[i].Type = "required"
		}
	}

	if params.Location != "" {
		graphReq.Location = &graphEventLocation{DisplayName: params.Location}
	}

	path := microsoftCalendarEventsBasePath(params.CalendarID)

	var resp graphEventResponse
	if err := a.conn.doRequest(ctx, http.MethodPost, path, req.Credentials, graphReq, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":       resp.ID,
		"subject":  resp.Subject,
		"start":    resp.Start.DateTime,
		"end":      resp.End.DateTime,
		"web_link": resp.WebLink,
	})
}
