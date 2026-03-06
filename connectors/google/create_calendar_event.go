package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCalendarEventAction implements connectors.Action for google.create_calendar_event.
// It creates an event via the Google Calendar API POST /calendars/{calendarId}/events.
type createCalendarEventAction struct {
	conn *GoogleConnector
}

// createCalendarEventParams is the user-facing parameter schema.
type createCalendarEventParams struct {
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	Attendees   []string `json:"attendees"`
	CalendarID  string   `json:"calendar_id"`
}

func (p *createCalendarEventParams) validate() error {
	if p.Summary == "" {
		return &connectors.ValidationError{Message: "missing required parameter: summary"}
	}
	if p.StartTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start_time"}
	}
	if p.EndTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: end_time"}
	}
	if _, err := time.Parse(time.RFC3339, p.StartTime); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("start_time must be RFC 3339 format: %v", err)}
	}
	if _, err := time.Parse(time.RFC3339, p.EndTime); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("end_time must be RFC 3339 format: %v", err)}
	}
	return nil
}

func (p *createCalendarEventParams) normalize() {
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}
}

// calendarEventRequest is the Google Calendar API request body for events.insert.
type calendarEventRequest struct {
	Summary     string                `json:"summary"`
	Description string                `json:"description,omitempty"`
	Start       calendarEventDateTime `json:"start"`
	End         calendarEventDateTime `json:"end"`
	Attendees   []calendarAttendee    `json:"attendees,omitempty"`
}

type calendarEventDateTime struct {
	DateTime string `json:"dateTime"`
}

type calendarAttendee struct {
	Email string `json:"email"`
}

// calendarEventResponse is the Google Calendar API response from events.insert.
type calendarEventResponse struct {
	ID          string `json:"id"`
	HTMLLink    string `json:"htmlLink"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	Description string `json:"description"`
}

// Execute creates a Google Calendar event and returns its metadata.
func (a *createCalendarEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCalendarEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	body := calendarEventRequest{
		Summary:     params.Summary,
		Description: params.Description,
		Start:       calendarEventDateTime{DateTime: params.StartTime},
		End:         calendarEventDateTime{DateTime: params.EndTime},
	}
	for _, email := range params.Attendees {
		body.Attendees = append(body.Attendees, calendarAttendee{Email: email})
	}

	var resp calendarEventResponse
	url := a.conn.calendarBaseURL + "/calendars/" + params.CalendarID + "/events"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, url, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":        resp.ID,
		"html_link": resp.HTMLLink,
		"status":    resp.Status,
	})
}
