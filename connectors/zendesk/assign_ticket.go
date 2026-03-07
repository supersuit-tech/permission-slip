package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// assignTicketAction implements connectors.Action for zendesk.assign_ticket.
// It assigns a ticket to an agent or group via PUT /tickets/{id}.json.
type assignTicketAction struct {
	conn *ZendeskConnector
}

type assignTicketParams struct {
	TicketID   int64  `json:"ticket_id"`
	AssigneeID *int64 `json:"assignee_id"`
	GroupID    *int64 `json:"group_id"`
}

func (p *assignTicketParams) validate() error {
	if !isValidZendeskID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if p.AssigneeID == nil && p.GroupID == nil {
		return &connectors.ValidationError{Message: "at least one of assignee_id or group_id must be provided"}
	}
	if p.AssigneeID != nil && *p.AssigneeID <= 0 {
		return &connectors.ValidationError{Message: "assignee_id must be a positive integer"}
	}
	if p.GroupID != nil && *p.GroupID <= 0 {
		return &connectors.ValidationError{Message: "group_id must be a positive integer"}
	}
	return nil
}

func (a *assignTicketAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params assignTicketParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	t := ticket{
		AssigneeID: params.AssigneeID,
		GroupID:    params.GroupID,
	}

	body := map[string]ticket{"ticket": t}
	var resp ticketResponse
	path := fmt.Sprintf("/tickets/%d.json", params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
