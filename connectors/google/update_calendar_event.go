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

// ParameterAliases maps common agent shorthand to the canonical parameter names.
// Agents sometimes send "start"/"end" instead of "start_time"/"end_time".
func (a *updateCalendarEventAction) ParameterAliases() map[string]string {
	return map[string]string{
		"start": "start_time",
		"end":   "end_time",
	}
}

// updateCalendarEventParams is the user-facing parameter schema.
type updateCalendarEventParams struct {
	EventID        string   `json:"event_id"`
	CalendarID     string   `json:"calendar_id"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	Attendees      []string `json:"attendees"`
	Location       string   `json:"location"`
	ClearAttendees bool     `json:"clear_attendees"`
}

func (p *updateCalendarEventParams) validate() error {
	if p.EventID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_id"}
	}
	hasUpdate := p.Summary != "" || p.Description != "" || p.Location != "" ||
		p.StartTime != "" || p.EndTime != "" || len(p.Attendees) > 0 || p.ClearAttendees
	if !hasUpdate {
		return &connectors.ValidationError{Message: "at least one field to update must be provided"}
	}
	if p.ClearAttendees && len(p.Attendees) > 0 {
		return &connectors.ValidationError{Message: "clear_attendees and attendees are mutually exclusive"}
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

	// Use map[string]any so we can include an explicit empty attendees array
	// when clear_attendees is true. Struct-based marshaling with omitempty
	// cannot distinguish "not provided" from "empty list".
	body := map[string]any{}
	if params.Summary != "" {
		body["summary"] = params.Summary
	}
	if params.Description != "" {
		body["description"] = params.Description
	}
	if params.Location != "" {
		body["location"] = params.Location
	}
	if params.StartTime != "" {
		body["start"] = calendarEventDateTime{DateTime: params.StartTime}
		body["end"] = calendarEventDateTime{DateTime: params.EndTime}
	}
	switch {
	case params.ClearAttendees:
		body["attendees"] = []calendarAttendee{}
	case len(params.Attendees) > 0:
		body["attendees"] = buildAttendees(params.Attendees)
	}

	var resp calendarEventResponse
	patchURL := a.conn.calendarBaseURL + "/calendars/" + url.PathEscape(params.CalendarID) + "/events/" + url.PathEscape(params.EventID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, patchURL, body, &resp); err != nil {
		return nil, err
	}

	result := map[string]string{
		"id":        resp.ID,
		"html_link": resp.HTMLLink,
		"status":    resp.Status,
		"updated":   resp.Updated,
	}
	if resp.Summary != "" {
		result["summary"] = resp.Summary
	}
	return connectors.JSONResult(result)
}
