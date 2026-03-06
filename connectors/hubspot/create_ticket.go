package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createTicketAction implements connectors.Action for hubspot.create_ticket.
// It creates a support ticket via POST /crm/v3/objects/tickets.
type createTicketAction struct {
	conn *HubSpotConnector
}

type createTicketParams struct {
	Subject        string            `json:"subject"`
	Content        string            `json:"content"`
	Pipeline       string            `json:"hs_pipeline"`
	PipelineStage  string            `json:"hs_pipeline_stage"`
	TicketPriority string            `json:"hs_ticket_priority"`
	Properties     map[string]string `json:"properties"`
}

func (p *createTicketParams) validate() error {
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if p.Pipeline == "" {
		return &connectors.ValidationError{Message: "missing required parameter: hs_pipeline"}
	}
	if p.PipelineStage == "" {
		return &connectors.ValidationError{Message: "missing required parameter: hs_pipeline_stage"}
	}
	return nil
}

func (p *createTicketParams) toAPIProperties() map[string]string {
	props := make(map[string]string)
	for k, v := range p.Properties {
		props[k] = v
	}
	props["subject"] = p.Subject
	props["hs_pipeline"] = p.Pipeline
	props["hs_pipeline_stage"] = p.PipelineStage
	if p.Content != "" {
		props["content"] = p.Content
	}
	if p.TicketPriority != "" {
		props["hs_ticket_priority"] = p.TicketPriority
	}
	return props
}

func (a *createTicketAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTicketParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.toAPIProperties()}
	var resp hubspotObjectResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/tickets", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
