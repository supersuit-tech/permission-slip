package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createTicketAction implements connectors.Action for zendesk.create_ticket.
// It creates a ticket via POST /tickets.json.
type createTicketAction struct {
	conn *ZendeskConnector
}

type createTicketParams struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Type        string `json:"type"`
	Tags        []string `json:"tags"`
	RequesterID *int64   `json:"requester_id"`
}

var validPriorities = map[string]bool{
	"urgent": true, "high": true, "normal": true, "low": true,
}

var validTicketTypes = map[string]bool{
	"problem": true, "incident": true, "question": true, "task": true,
}

func (p *createTicketParams) validate() error {
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	if p.Priority != "" && !validPriorities[p.Priority] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid priority %q: must be urgent, high, normal, or low", p.Priority)}
	}
	if p.Type != "" && !validTicketTypes[p.Type] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid type %q: must be problem, incident, question, or task", p.Type)}
	}
	return nil
}

func (a *createTicketAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTicketParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	t := ticket{
		Subject:     params.Subject,
		Description: params.Description,
		Tags:        params.Tags,
		RequesterID: params.RequesterID,
	}
	if params.Priority != "" {
		t.Priority = params.Priority
	}
	if params.Type != "" {
		t.Type = params.Type
	}

	body := map[string]ticket{"ticket": t}
	var resp ticketResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tickets.json", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
