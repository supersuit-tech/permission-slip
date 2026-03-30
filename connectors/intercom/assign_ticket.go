package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// assignTicketAction implements connectors.Action for intercom.assign_ticket.
// It assigns a ticket to an admin or team via PUT /tickets/{id}.
type assignTicketAction struct {
	conn *IntercomConnector
}

type assignTicketParams struct {
	TicketID   string `json:"ticket_id"`
	AssigneeID string `json:"assignee_id"`
}

func (p *assignTicketParams) validate() error {
	if !isValidIntercomID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if !isValidIntercomID(p.AssigneeID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: assignee_id"}
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

	body := map[string]any{
		"assignment": map[string]string{
			"admin_id": params.AssigneeID,
		},
	}

	var resp intercomTicket
	path := "/tickets/" + url.PathEscape(params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
