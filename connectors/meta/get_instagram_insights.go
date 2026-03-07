package meta

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getInstagramInsightsAction implements connectors.Action for meta.get_instagram_insights.
// It retrieves account-level insights via GET /{ig_account_id}/insights.
type getInstagramInsightsAction struct {
	conn *MetaConnector
}

type getInstagramInsightsParams struct {
	InstagramAccountID string `json:"instagram_account_id"`
	Metric             string `json:"metric,omitempty"`
	Period             string `json:"period,omitempty"`
}

var validMetrics = map[string]bool{
	"impressions":   true,
	"reach":         true,
	"profile_views": true,
}

var validPeriods = map[string]bool{
	"day":     true,
	"week":    true,
	"days_28": true,
}

func (p *getInstagramInsightsParams) validate() error {
	if p.InstagramAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: instagram_account_id"}
	}
	if p.Metric != "" && !validMetrics[p.Metric] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid metric %q: must be impressions, reach, or profile_views", p.Metric),
		}
	}
	if p.Period != "" && !validPeriods[p.Period] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid period %q: must be day, week, or days_28", p.Period),
		}
	}
	return nil
}

type insightsResponse struct {
	Data []insightData `json:"data"`
}

type insightData struct {
	Name   string         `json:"name"`
	Period string         `json:"period"`
	Values []insightValue `json:"values"`
	Title  string         `json:"title"`
}

type insightValue struct {
	Value   int    `json:"value"`
	EndTime string `json:"end_time"`
}

func (a *getInstagramInsightsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getInstagramInsightsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	metric := params.Metric
	if metric == "" {
		metric = "impressions"
	}
	period := params.Period
	if period == "" {
		period = "day"
	}

	url := fmt.Sprintf("%s/%s/insights?metric=%s&period=%s",
		a.conn.baseURL, params.InstagramAccountID, metric, period)

	var resp insightsResponse
	if err := a.conn.doGet(ctx, req.Credentials, url, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
