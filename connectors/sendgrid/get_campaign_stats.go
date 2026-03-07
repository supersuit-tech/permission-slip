package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getCampaignStatsAction implements connectors.Action for sendgrid.get_campaign_stats.
// It retrieves analytics for a single send campaign via
// GET /marketing/singlesends/{id}.
type getCampaignStatsAction struct {
	conn *SendGridConnector
}

type getCampaignStatsParams struct {
	SingleSendID string `json:"singlesend_id"`
}

func (p *getCampaignStatsParams) validate() error {
	if p.SingleSendID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: singlesend_id"}
	}
	return nil
}

// Execute retrieves campaign stats including send count, opens, clicks, etc.
func (a *getCampaignStatsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getCampaignStatsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/marketing/singlesends/" + url.PathEscape(params.SingleSendID)

	var resp struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Stats  *struct {
			Requests         int `json:"requests"`
			Delivered        int `json:"delivered"`
			Opens            int `json:"opens"`
			UniqueOpens      int `json:"unique_opens"`
			Clicks           int `json:"clicks"`
			UniqueClicks     int `json:"unique_clicks"`
			Bounces          int `json:"bounces"`
			SpamReports      int `json:"spam_reports"`
			Unsubscribes     int `json:"unsubscribes"`
		} `json:"send_stats"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"singlesend_id": resp.ID,
		"name":          resp.Name,
		"status":        resp.Status,
	}
	if resp.Stats != nil {
		result["stats"] = map[string]int{
			"requests":      resp.Stats.Requests,
			"delivered":     resp.Stats.Delivered,
			"opens":         resp.Stats.Opens,
			"unique_opens":  resp.Stats.UniqueOpens,
			"clicks":        resp.Stats.Clicks,
			"unique_clicks": resp.Stats.UniqueClicks,
			"bounces":       resp.Stats.Bounces,
			"spam_reports":  resp.Stats.SpamReports,
			"unsubscribes":  resp.Stats.Unsubscribes,
		}
	}

	return connectors.JSONResult(result)
}
