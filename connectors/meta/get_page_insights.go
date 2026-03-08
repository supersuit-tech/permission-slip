package meta

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getPageInsightsAction implements connectors.Action for meta.get_page_insights.
// It retrieves Facebook Page insights via GET /{page_id}/insights.
type getPageInsightsAction struct {
	conn *MetaConnector
}

type getPageInsightsParams struct {
	PageID string `json:"page_id"`
	Metric string `json:"metric,omitempty"`
	Period string `json:"period,omitempty"`
	Since  int64  `json:"since,omitempty"`
	Until  int64  `json:"until,omitempty"`
}

var validPageInsightMetrics = map[string]bool{
	"page_impressions":                   true,
	"page_impressions_unique":            true,
	"page_engaged_users":                 true,
	"page_post_engagements":              true,
	"page_fan_adds":                      true,
	"page_fan_removes":                   true,
	"page_views_total":                   true,
	"page_reach":                         true,
}

var validPageInsightPeriods = map[string]bool{
	"day":     true,
	"week":    true,
	"days_28": true,
	"month":   true,
}

func (p *getPageInsightsParams) validate() error {
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	if !isValidGraphID(p.PageID) {
		return &connectors.ValidationError{Message: "page_id contains invalid characters"}
	}
	if p.Metric != "" && !validPageInsightMetrics[p.Metric] {
		return &connectors.ValidationError{Message: "invalid metric — must be one of: page_impressions, page_impressions_unique, page_engaged_users, page_post_engagements, page_fan_adds, page_fan_removes, page_views_total, page_reach"}
	}
	if p.Period != "" && !validPageInsightPeriods[p.Period] {
		return &connectors.ValidationError{Message: "invalid period — must be one of: day, week, days_28, month"}
	}
	return nil
}

type pageInsightsResponse struct {
	Data []pageInsightDataPoint `json:"data"`
}

type pageInsightDataPoint struct {
	ID     string              `json:"id"`
	Name   string              `json:"name"`
	Period string              `json:"period"`
	Values []pageInsightValue  `json:"values"`
	Title  string              `json:"title"`
}

type pageInsightValue struct {
	Value     interface{} `json:"value"`
	EndTime   string      `json:"end_time"`
}

func (a *getPageInsightsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPageInsightsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	metric := params.Metric
	if metric == "" {
		metric = "page_impressions"
	}
	period := params.Period
	if period == "" {
		period = "day"
	}

	reqURL := fmt.Sprintf("%s/%s/insights?metric=%s&period=%s",
		a.conn.baseURL, params.PageID, metric, period)

	if params.Since > 0 {
		reqURL += fmt.Sprintf("&since=%d", params.Since)
	}
	if params.Until > 0 {
		reqURL += fmt.Sprintf("&until=%d", params.Until)
	}

	var resp pageInsightsResponse
	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
