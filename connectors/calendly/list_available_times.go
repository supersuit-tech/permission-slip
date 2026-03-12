package calendly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listAvailableTimesAction implements connectors.Action for calendly.list_available_times.
// It lists available time slots via GET /event_type_available_times.
type listAvailableTimesAction struct {
	conn *CalendlyConnector
}

// ParameterAliases maps common agent shorthand to the canonical parameter names.
// Agents sometimes send "start"/"end" instead of "start_time"/"end_time".
func (a *listAvailableTimesAction) ParameterAliases() map[string]string {
	return map[string]string{
		"start": "start_time",
		"end":   "end_time",
	}
}

type listAvailableTimesParams struct {
	EventTypeURI string `json:"event_type_uri"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}

func (p *listAvailableTimesParams) validate() error {
	if p.EventTypeURI == "" {
		return &connectors.ValidationError{Message: "missing required parameter: event_type_uri"}
	}
	if p.StartTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start_time"}
	}
	if p.EndTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: end_time"}
	}
	return nil
}

type calendlyAvailableTimesResponse struct {
	Collection []calendlyAvailableTime `json:"collection"`
}

type calendlyAvailableTime struct {
	Status        string `json:"status"`
	InviteesRemaining int    `json:"invitees_remaining"`
	StartTime     string `json:"start_time"`
	SchedulingURL string `json:"scheduling_url"`
}

func (a *listAvailableTimesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listAvailableTimesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("event_type", params.EventTypeURI)
	q.Set("start_time", params.StartTime)
	q.Set("end_time", params.EndTime)

	var resp calendlyAvailableTimesResponse
	reqURL := a.conn.baseURL + "/event_type_available_times?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, reqURL, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"total_available_times": len(resp.Collection),
		"available_times":       resp.Collection,
	})
}
