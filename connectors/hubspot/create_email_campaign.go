package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createEmailCampaignAction implements connectors.Action for hubspot.create_email_campaign.
// It creates a marketing email via POST /marketing/v3/emails, and optionally sends it.
type createEmailCampaignAction struct {
	conn *HubSpotConnector
}

type createEmailCampaignParams struct {
	Name    string   `json:"name"`
	Subject string   `json:"subject"`
	Content string   `json:"content"`
	ListIDs []string `json:"list_ids"`
	SendNow bool     `json:"send_now"`
}

func (p *createEmailCampaignParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	if p.SendNow && len(p.ListIDs) == 0 {
		return &connectors.ValidationError{Message: "list_ids is required when send_now is true — specify at least one contact list to send to"}
	}
	for i, id := range p.ListIDs {
		if !isValidHubSpotID(id) {
			return &connectors.ValidationError{Message: fmt.Sprintf("list_ids[%d]: must be a numeric HubSpot ID", i)}
		}
	}
	return nil
}

// emailCampaignRequest is the request body for the marketing email API.
type emailCampaignRequest struct {
	Name    string   `json:"name"`
	Subject string   `json:"subject"`
	Content string   `json:"content"`
	ListIDs []string `json:"listIds,omitempty"`
	SendNow bool     `json:"sendNow"`
}

// emailCampaignResponse captures the response from the marketing email API.
type emailCampaignResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subject string `json:"subject"`
	State   string `json:"state"`
}

func (a *createEmailCampaignAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createEmailCampaignParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := emailCampaignRequest{
		Name:    params.Name,
		Subject: params.Subject,
		Content: params.Content,
		ListIDs: params.ListIDs,
		SendNow: params.SendNow,
	}

	var resp emailCampaignResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/marketing/v3/emails", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
