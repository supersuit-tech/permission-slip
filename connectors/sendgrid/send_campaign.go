package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sendCampaignAction implements connectors.Action for sendgrid.send_campaign.
// It creates and immediately sends a single send campaign via the SendGrid v3
// Marketing Campaigns API.
type sendCampaignAction struct {
	conn *SendGridConnector
}

// Execute creates a single send campaign and immediately sends it.
func (a *sendCampaignAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var fields campaignFields
	if err := json.Unmarshal(req.Parameters, &fields); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := fields.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create the single send
	createBody := buildSingleSendBody(&fields)
	var createResp struct {
		ID string `json:"id"`
	}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, "/marketing/singlesends", createBody, &createResp); err != nil {
		return nil, err
	}

	// Step 2: Send it immediately
	var sendResp struct {
		Status string `json:"status"`
	}
	sendPath := "/marketing/singlesends/" + url.PathEscape(createResp.ID) + "/schedule"
	sendBody := map[string]string{"send_at": "now"}
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPut, sendPath, sendBody, &sendResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"singlesend_id": createResp.ID,
		"status":        sendResp.Status,
	})
}
