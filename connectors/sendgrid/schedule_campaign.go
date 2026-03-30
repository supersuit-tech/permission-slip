package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// scheduleCampaignAction implements connectors.Action for sendgrid.schedule_campaign.
// It creates a single send campaign and schedules it for future delivery.
type scheduleCampaignAction struct {
	conn *SendGridConnector
}

type scheduleCampaignParams struct {
	campaignFields
	SendAt string `json:"send_at"`
}

func (p *scheduleCampaignParams) validate() error {
	if err := p.campaignFields.validate(); err != nil {
		return err
	}
	if p.SendAt == "" {
		return &connectors.ValidationError{Message: "missing required parameter: send_at"}
	}
	sendTime, err := time.Parse(time.RFC3339, p.SendAt)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("send_at must be a valid ISO 8601 timestamp (e.g. 2026-03-15T10:00:00Z), got %q", p.SendAt)}
	}
	if sendTime.Before(time.Now()) {
		return &connectors.ValidationError{Message: "send_at must be in the future"}
	}
	return nil
}

// Execute creates a single send campaign and schedules it for the specified time.
func (a *scheduleCampaignAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params scheduleCampaignParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create the single send
	createBody := buildSingleSendBody(&params.campaignFields)
	var createResp struct {
		ID string `json:"id"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, "/marketing/singlesends", createBody, &createResp); err != nil {
		return nil, err
	}

	// Step 2: Schedule it
	var scheduleResp struct {
		Status string `json:"status"`
		SendAt string `json:"send_at"`
	}
	schedulePath := "/marketing/singlesends/" + url.PathEscape(createResp.ID) + "/schedule"
	scheduleBody := map[string]string{"send_at": params.SendAt}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPut, schedulePath, scheduleBody, &scheduleResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"singlesend_id": createResp.ID,
		"status":        scheduleResp.Status,
		"send_at":       scheduleResp.SendAt,
	})
}
