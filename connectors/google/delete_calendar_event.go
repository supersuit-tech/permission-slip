package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteCalendarEventAction implements connectors.Action for google.delete_calendar_event.
// It deletes an event via the Google Calendar API DELETE /calendars/{calendarId}/events/{eventId}.
type deleteCalendarEventAction struct {
	conn *GoogleConnector
}

// deleteCalendarEventParams is the user-facing parameter schema.
type deleteCalendarEventParams struct {
	EventID       string `json:"event_id"`
	CalendarID    string `json:"calendar_id"`
	Scope         string `json:"scope"`
	InstanceStart string `json:"instance_start"`
}

func (p *deleteCalendarEventParams) validate() error {
	if p.EventID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_id"}
	}
	return validateScopeAgainstEventID(p.Scope, p.EventID, p.InstanceStart)
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

	if params.Scope == calendarScopeThisAndFollowing {
		return a.executeThisAndFollowing(ctx, req, params)
	}

	target, err := a.conn.resolveCalendarEventTarget(ctx, req.Credentials, params.CalendarID, params.EventID, params.Scope, params.InstanceStart)
	if err != nil {
		return nil, err
	}

	deleteURL := calendarEventPath(a.conn.calendarBaseURL, params.CalendarID, target.TargetID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, deleteURL, nil, nil); err != nil {
		return nil, err
	}

	out := map[string]string{
		"event_id":    target.TargetID,
		"calendar_id": params.CalendarID,
		"status":      "deleted",
	}
	if params.Scope != "" {
		out["scope"] = params.Scope
	}
	return connectors.JSONResult(out)
}

func (a *deleteCalendarEventAction) executeThisAndFollowing(ctx context.Context, req connectors.ActionRequest, params deleteCalendarEventParams) (*connectors.ActionResult, error) {
	target, err := a.conn.resolveCalendarEventTarget(ctx, req.Credentials, params.CalendarID, params.EventID, params.Scope, params.InstanceStart)
	if err != nil {
		return nil, err
	}

	// Non-recurring or first instance: delete the whole event/series.
	if target.Kind == calendarEventKindSingle || target.Master == nil || target.Instance == nil ||
		isFirstSeriesInstance(target.Master, target.Instance) {
		deleteID := target.TargetID
		if target.Master != nil {
			deleteID = target.Master.ID
		}
		deleteURL := calendarEventPath(a.conn.calendarBaseURL, params.CalendarID, deleteID)
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodDelete, deleteURL, nil, nil); err != nil {
			return nil, err
		}
		return connectors.JSONResult(map[string]string{
			"event_id":    deleteID,
			"calendar_id": params.CalendarID,
			"status":      "deleted",
			"scope":       calendarScopeThisAndFollowing,
		})
	}

	if err := a.conn.truncateSeriesBeforeInstance(ctx, req.Credentials, params.CalendarID, target.Master, target.Instance); err != nil {
		return nil, err
	}
	return connectors.JSONResult(map[string]string{
		"event_id":          target.Instance.ID,
		"calendar_id":       params.CalendarID,
		"status":            "truncated",
		"scope":             calendarScopeThisAndFollowing,
		"original_event_id": target.Master.ID,
	})
}
