package sendgrid

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// scheduleCampaignAction implements connectors.Action for sendgrid.schedule_campaign.
// It creates a single send campaign and schedules it for future delivery.
type scheduleCampaignAction struct {
	conn *SendGridConnector
}

type scheduleCampaignParams struct {
	Name               string   `json:"name"`
	Subject            string   `json:"subject"`
	HTMLContent        string   `json:"html_content"`
	PlainContent       string   `json:"plain_content"`
	ListIDs            []string `json:"list_ids"`
	SenderID           int      `json:"sender_id"`
	SendAt             string   `json:"send_at"`
	SuppressionGroupID int      `json:"suppression_group_id,omitempty"`
}

func (p *scheduleCampaignParams) validate() error {
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
	campaignParams := &sendCampaignParams{
		Name:               params.Name,
		Subject:            params.Subject,
		HTMLContent:        params.HTMLContent,
		PlainContent:       params.PlainContent,
		ListIDs:            params.ListIDs,
		SenderID:           params.SenderID,
		SuppressionGroupID: params.SuppressionGroupID,
	}
	createBody := buildSingleSendBody(campaignParams)
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
