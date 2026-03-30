package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getAnalyticsAction implements connectors.Action for hubspot.get_analytics.
// It fetches analytics data via GET /analytics/v2/reports/{object_type}/{time_period}.
type getAnalyticsAction struct {
	conn *HubSpotConnector
}

type getAnalyticsParams struct {
	ObjectType string `json:"object_type"`
	TimePeriod string `json:"time_period"`
	Start      string `json:"start"`
	End        string `json:"end"`
}

var validAnalyticsObjectTypes = map[string]bool{
	"contacts":  true,
	"deals":     true,
	"companies": true,
	"tickets":   true,
}

var validAnalyticsTimePeriods = map[string]bool{
	"total":   true,
	"daily":   true,
	"weekly":  true,
	"monthly": true,
}

func (p *getAnalyticsParams) validate() error {
	if p.ObjectType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: object_type"}
	}
	if !validAnalyticsObjectTypes[p.ObjectType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid object_type %q: must be contacts, deals, companies, or tickets", p.ObjectType)}
	}
	if p.TimePeriod == "" {
		return &connectors.ValidationError{Message: "missing required parameter: time_period"}
	}
	if !validAnalyticsTimePeriods[p.TimePeriod] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid time_period %q: must be total, daily, weekly, or monthly", p.TimePeriod)}
	}
	return nil
}

// analyticsResponse captures the response from the analytics API.
type analyticsResponse struct {
	Breakdowns []json.RawMessage `json:"breakdowns"`
	Total      json.RawMessage   `json:"total,omitempty"`
	Offset     json.RawMessage   `json:"offset,omitempty"`
}

func (a *getAnalyticsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getAnalyticsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/analytics/v2/reports/%s/%s",
		url.PathEscape(params.ObjectType),
		url.PathEscape(params.TimePeriod),
	)

	startNorm, err := connectors.NormalizeHubSpotAnalyticsTimeParam(params.Start)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid start: %v", err)}
	}
	endNorm, err := connectors.NormalizeHubSpotAnalyticsTimeParam(params.End)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid end: %v", err)}
	}

	query := url.Values{}
	if startNorm != "" {
		query.Set("start", startNorm)
	}
	if endNorm != "" {
		query.Set("end", endNorm)
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var resp analyticsResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
