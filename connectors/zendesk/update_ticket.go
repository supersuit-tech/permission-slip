package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateTicketAction implements connectors.Action for zendesk.update_ticket.
// It updates a ticket's status via PUT /tickets/{id}.json.
type updateTicketAction struct {
	conn *ZendeskConnector
}

type updateTicketParams struct {
	TicketID int64  `json:"ticket_id"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Type     string `json:"type"`
}

var validStatuses = map[string]bool{
	"new": true, "open": true, "pending": true, "hold": true, "solved": true, "closed": true,
}

func (p *updateTicketParams) validate() error {
	if !isValidZendeskID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if p.Status == "" && p.Priority == "" && p.Type == "" {
		return &connectors.ValidationError{Message: "at least one of status, priority, or type must be provided"}
	}
	if p.Status != "" && !validStatuses[p.Status] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid status %q: must be new, open, pending, hold, solved, or closed", p.Status)}
	}
	if p.Priority != "" && !validPriorities[p.Priority] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid priority %q: must be urgent, high, normal, or low", p.Priority)}
	}
	if p.Type != "" && !validTicketTypes[p.Type] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid type %q: must be problem, incident, question, or task", p.Type)}
	}
	return nil
}

func (a *updateTicketAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateTicketParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	t := ticket{}
	if params.Status != "" {
		t.Status = params.Status
	}
	if params.Priority != "" {
		t.Priority = params.Priority
	}
	if params.Type != "" {
		t.Type = params.Type
	}

	body := map[string]ticket{"ticket": t}
	var resp ticketResponse
	path := fmt.Sprintf("/tickets/%d.json", params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
