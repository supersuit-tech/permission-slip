package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	Scope          string   `json:"scope"`
	InstanceStart  string   `json:"instance_start"`
	Recurrence     []string `json:"recurrence"`
}

func (p *updateCalendarEventParams) validate() error {
	if p.EventID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_id"}
	}
	hasUpdate := p.Summary != "" || p.Description != "" || p.Location != "" ||
		p.StartTime != "" || p.EndTime != "" || len(p.Attendees) > 0 || p.ClearAttendees ||
		len(p.Recurrence) > 0
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
	if len(p.Recurrence) > 0 {
		if err := validateRecurrence(p.Recurrence); err != nil {
			return err
		}
		if p.Scope == calendarScopeInstance {
			return &connectors.ValidationError{
				Message: "recurrence can only be updated on the series master (scope=series) or when splitting with scope=this_and_following",
			}
		}
		if p.Scope == "" && looksLikeInstanceEventID(p.EventID) {
			return &connectors.ValidationError{
				Message: "recurrence cannot be updated on an expanded instance id; pass the series master event_id with scope=series",
			}
		}
	}
	return validateScopeAgainstEventID(p.Scope, p.EventID, p.InstanceStart)
}

func (p *updateCalendarEventParams) normalize() {
	if p.CalendarID == "" {
		p.CalendarID = "primary"
	}
}

func (p *updateCalendarEventParams) patchBody() map[string]any {
	// Use map[string]any so we can include an explicit empty attendees array
	// when clear_attendees is true. Struct-based marshaling with omitempty
	// cannot distinguish "not provided" from "empty list".
	body := map[string]any{}
	if p.Summary != "" {
		body["summary"] = p.Summary
	}
	if p.Description != "" {
		body["description"] = p.Description
	}
	if p.Location != "" {
		body["location"] = p.Location
	}
	if p.StartTime != "" {
		body["start"] = calendarEventDateTime{DateTime: p.StartTime}
		body["end"] = calendarEventDateTime{DateTime: p.EndTime}
	}
	switch {
	case p.ClearAttendees:
		body["attendees"] = []calendarAttendee{}
	case len(p.Attendees) > 0:
		body["attendees"] = buildAttendees(p.Attendees)
	}
	if len(p.Recurrence) > 0 {
		body["recurrence"] = p.Recurrence
	}
	return body
}

func updateCalendarEventResult(resp calendarEventResponse, extra map[string]string) map[string]string {
	result := map[string]string{
		"id":        resp.ID,
		"html_link": resp.HTMLLink,
		"status":    resp.Status,
		"updated":   resp.Updated,
	}
	if resp.Summary != "" {
		result["summary"] = resp.Summary
	}
	for k, v := range extra {
		if v != "" {
			result[k] = v
		}
	}
	return result
}

// Execute patches an existing Google Calendar event and returns its updated metadata.
func (a *updateCalendarEventAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateCalendarEventParams
	if err := json.Unmarshal(normalizeCalendarTimeParams(req.Parameters), &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	body := params.patchBody()

	if params.Scope == calendarScopeThisAndFollowing {
		return a.executeThisAndFollowing(ctx, req, params, body)
	}

	target, err := a.conn.resolveCalendarEventTarget(ctx, req.Credentials, params.CalendarID, params.EventID, params.Scope, params.InstanceStart)
	if err != nil {
		return nil, err
	}

	var resp calendarEventResponse
	patchURL := calendarEventPath(a.conn.calendarBaseURL, params.CalendarID, target.TargetID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, patchURL, body, &resp); err != nil {
		return nil, err
	}

	extra := map[string]string{}
	if params.Scope != "" {
		extra["scope"] = params.Scope
	}
	return connectors.JSONResult(updateCalendarEventResult(resp, extra))
}

func (a *updateCalendarEventAction) executeThisAndFollowing(ctx context.Context, req connectors.ActionRequest, params updateCalendarEventParams, body map[string]any) (*connectors.ActionResult, error) {
	target, err := a.conn.resolveCalendarEventTarget(ctx, req.Credentials, params.CalendarID, params.EventID, params.Scope, params.InstanceStart)
	if err != nil {
		return nil, err
	}

	// Single (non-recurring) events have no tail to split — PATCH in place.
	if target.Kind == calendarEventKindSingle || target.Master == nil || target.Instance == nil {
		var resp calendarEventResponse
		patchURL := calendarEventPath(a.conn.calendarBaseURL, params.CalendarID, target.TargetID)
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, patchURL, body, &resp); err != nil {
			return nil, err
		}
		return connectors.JSONResult(updateCalendarEventResult(resp, map[string]string{
			"scope": calendarScopeThisAndFollowing,
		}))
	}

	if isFirstSeriesInstance(target.Master, target.Instance) {
		var resp calendarEventResponse
		patchURL := calendarEventPath(a.conn.calendarBaseURL, params.CalendarID, target.Master.ID)
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPatch, patchURL, body, &resp); err != nil {
			return nil, err
		}
		return connectors.JSONResult(updateCalendarEventResult(resp, map[string]string{
			"scope":             calendarScopeThisAndFollowing,
			"original_event_id": target.Master.ID,
		}))
	}

	// Count remaining instances on the original series before we rewrite UNTIL.
	newRecurrence, err := a.conn.recurrenceForSplitSeries(ctx, req.Credentials, params.CalendarID, target.Master, target.Instance, params.Recurrence)
	if err != nil {
		return nil, err
	}
	if err := a.conn.truncateSeriesBeforeInstance(ctx, req.Credentials, params.CalendarID, target.Master, target.Instance); err != nil {
		return nil, err
	}
	insertBody := newSeriesBodyFromMaster(target.Master, target.Instance, body, newRecurrence)

	var resp calendarEventResponse
	insertURL := a.conn.calendarBaseURL + "/calendars/" + url.PathEscape(params.CalendarID) + "/events"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, insertURL, insertBody, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(updateCalendarEventResult(resp, map[string]string{
		"scope":             calendarScopeThisAndFollowing,
		"original_event_id": target.Master.ID,
	}))
}
