package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendCampaignAction implements connectors.Action for sendgrid.send_campaign.
// It creates and immediately sends a single send campaign via the SendGrid v3
// Marketing Campaigns API.
type sendCampaignAction struct {
	conn *SendGridConnector
}

type sendCampaignParams struct {
	Name               string   `json:"name"`
	Subject            string   `json:"subject"`
	HTMLContent        string   `json:"html_content"`
	PlainContent       string   `json:"plain_content"`
	ListIDs            []string `json:"list_ids"`
	SenderID           int      `json:"sender_id"`
	SuppressionGroupID int      `json:"suppression_group_id,omitempty"`
}

func (p *sendCampaignParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if len(p.Name) > 100 {
		return &connectors.ValidationError{Message: fmt.Sprintf("name exceeds maximum length of 100 characters (got %d)", len(p.Name))}
	}
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if len(p.Subject) > 998 {
		return &connectors.ValidationError{Message: fmt.Sprintf("subject exceeds maximum length of 998 characters (got %d)", len(p.Subject))}
	}
	if len(p.ListIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: list_ids (must contain at least one list ID)"}
	}
	if p.SenderID == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: sender_id"}
	}
	if p.HTMLContent == "" && p.PlainContent == "" {
		return &connectors.ValidationError{Message: "at least one of html_content or plain_content must be provided"}
	}
	return nil
}

// Execute creates a single send campaign and immediately sends it.
func (a *sendCampaignAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendCampaignParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create the single send
	createBody := buildSingleSendBody(&params)
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

// buildSingleSendBody constructs the JSON body for creating a single send.
// Scheduling is handled separately via the /schedule endpoint.
func buildSingleSendBody(params *sendCampaignParams) map[string]any {
	body := map[string]any{
		"name": params.Name,
		"email_config": map[string]any{
			"subject":       params.Subject,
			"sender_id":     params.SenderID,
			"html_content":  params.HTMLContent,
			"plain_content": params.PlainContent,
		},
		"send_to": map[string]any{
			"list_ids": params.ListIDs,
		},
	}
	if params.SuppressionGroupID != 0 {
		emailConfig := body["email_config"].(map[string]any)
		emailConfig["suppression_group_id"] = params.SuppressionGroupID
	}
	return body
}
