package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteCalendarEventAction implements connectors.Action for google.delete_calendar_event.
// It deletes an event via the Google Calendar API DELETE /calendars/{calendarId}/events/{eventId}.
type deleteCalendarEventAction struct {
	conn *GoogleConnector
}

// deleteCalendarEventParams is the user-facing parameter schema.
type deleteCalendarEventParams struct {
	EventID    string `json:"event_id"`
	CalendarID string `json:"calendar_id"`
}

func (p *deleteCalendarEventParams) validate() error {
	if p.EventID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_id"}
	}
	return nil
}

func (p *deleteCalendarEventParams) normalize() {
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}
}

// Execute deletes a Google Calendar event.
func (a *deleteCalendarEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteCalendarEventParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	deleteURL := a.conn.calendarBaseURL + "/calendars/" + url.PathEscape(params.CalendarID) + "/events/" + url.PathEscape(params.EventID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, deleteURL, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"event_id":    params.EventID,
		"calendar_id": params.CalendarID,
		"status":      "deleted",
	})
}
