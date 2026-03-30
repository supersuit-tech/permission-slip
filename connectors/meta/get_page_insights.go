package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	Since  string `json:"since,omitempty"`
	Until  string `json:"until,omitempty"`
}

var validPageInsightMetrics = map[string]bool{
	"page_impressions":        true,
	"page_impressions_unique": true,
	"page_engaged_users":      true,
	"page_post_engagements":   true,
	"page_fan_adds":           true,
	"page_fan_removes":        true,
	"page_views_total":        true,
	"page_reach":              true,
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
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Period string             `json:"period"`
	Values []pageInsightValue `json:"values"`
	Title  string             `json:"title"`
}

type pageInsightValue struct {
	Value   interface{} `json:"value"`
	EndTime string      `json:"end_time"`
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

	q := url.Values{}
	q.Set("metric", metric)
	q.Set("period", period)
	sinceUnix, err := connectors.ParseUnixTimestampOrRFC3339(params.Since)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid since: %v", err)}
	}
	untilUnix, err := connectors.ParseUnixTimestampOrRFC3339(params.Until)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid until: %v", err)}
	}
	if sinceUnix > 0 {
		q.Set("since", strconv.FormatInt(sinceUnix, 10))
	}
	if untilUnix > 0 {
		q.Set("until", strconv.FormatInt(untilUnix, 10))
	}

	reqURL := fmt.Sprintf("%s/%s/insights?%s", a.conn.baseURL, params.PageID, q.Encode())

	var resp pageInsightsResponse
	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
