package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateTicketAction implements connectors.Action for intercom.update_ticket.
// It updates a ticket's state via PUT /tickets/{id}.
type updateTicketAction struct {
	conn *IntercomConnector
}

type updateTicketParams struct {
	TicketID   string       `json:"ticket_id"`
	State      string       `json:"state"`
	Title      string       `json:"title"`
	Attributes []ticketAttr `json:"attributes"`
}

var validStates = map[string]bool{
	"submitted": true, "in_progress": true, "waiting_on_customer": true, "resolved": true,
}

func (p *updateTicketParams) validate() error {
	if !isValidIntercomID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if p.State == "" && p.Title == "" && len(p.Attributes) == 0 {
		return &connectors.ValidationError{Message: "at least one of state, title, or attributes must be provided"}
	}
	if p.State != "" && !validStates[p.State] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid state %q: must be submitted, in_progress, waiting_on_customer, or resolved", p.State)}
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

	body := make(map[string]any)
	if params.State != "" {
		body["state"] = params.State
	}
	if params.Title != "" {
		body["title"] = params.Title
	}
	if len(params.Attributes) > 0 {
		attrs := make(map[string]string, len(params.Attributes))
		for _, a := range params.Attributes {
			attrs[a.Name] = a.Value
		}
		body["ticket_attributes"] = attrs
	}

	var resp intercomTicket
	path := "/tickets/" + url.PathEscape(params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
