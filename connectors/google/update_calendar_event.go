package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateCalendarEventAction implements connectors.Action for google.update_calendar_event.
// It patches an existing event via the Google Calendar API PATCH /calendars/{calendarId}/events/{eventId}.
type updateCalendarEventAction struct {
	conn *GoogleConnector
}

// updateCalendarEventParams is the user-facing parameter schema.
type updateCalendarEventParams struct {
	EventID     string   `json:"event_id"`
	CalendarID  string   `json:"calendar_id"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	StartTime   string   `json:"start_time"`
	EndTime     string   `json:"end_time"`
	Attendees   []string `json:"attendees"`
	Location    string   `json:"location"`
}

func (p *updateCalendarEventParams) validate() error {
	if p.EventID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_id"}
	}
	if p.StartTime == "" && p.EndTime == "" && p.Summary == "" && p.Description == "" && p.Location == "" && len(p.Attendees) == 0 {
		return &connectors.ValidationError{Message: "at least one field to update must be provided"}
	}
	if (p.StartTime != "") != (p.EndTime != "") {
		return &connectors.ValidationError{Message: "start_time and end_time must both be provided when updating event time"}
	}
	if p.StartTime != "" && p.EndTime != "" {
		if err := validateTimeRange(p.StartTime, p.EndTime); err != nil {
			return err
		}
	}
	return nil
}

func (p *updateCalendarEventParams) normalize() {
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}
}

// calendarEventPatchRequest is the Google Calendar API request body for events.patch.
// All fields are pointers so omitempty works correctly for partial updates.
type calendarEventPatchRequest struct {
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Location    string                 `json:"location,omitempty"`
	Start       *calendarEventDateTime `json:"start,omitempty"`
	End         *calendarEventDateTime `json:"end,omitempty"`
	Attendees   []calendarAttendee     `json:"attendees,omitempty"`
}

// Execute patches an existing Google Calendar event and returns its updated metadata.
func (a *updateCalendarEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateCalendarEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	body := calendarEventPatchRequest{
		Summary:     params.Summary,
		Description: params.Description,
		Location:    params.Location,
		Attendees:   buildAttendees(params.Attendees),
	}
	if params.StartTime != "" {
		body.Start = &calendarEventDateTime{DateTime: params.StartTime}
		body.End = &calendarEventDateTime{DateTime: params.EndTime}
	}

	var resp calendarEventResponse
	patchURL := a.conn.calendarBaseURL + "/calendars/" + url.PathEscape(params.CalendarID) + "/events/" + url.PathEscape(params.EventID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, patchURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":        resp.ID,
		"html_link": resp.HTMLLink,
		"status":    resp.Status,
		"updated":   resp.Updated,
	})
}
