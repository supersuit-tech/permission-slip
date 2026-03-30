package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createTicketAction implements connectors.Action for intercom.create_ticket.
// It creates a ticket via POST /tickets.
type createTicketAction struct {
	conn *IntercomConnector
}

type createTicketParams struct {
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	TicketTypeID string       `json:"ticket_type_id"`
	ContactID    string       `json:"contact_id"`
	Attributes   []ticketAttr `json:"attributes"`
}

func (p *createTicketParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if p.TicketTypeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ticket_type_id"}
	}
	if p.ContactID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: contact_id"}
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

	body := map[string]any{
		"ticket_type_id": params.TicketTypeID,
		"title":          params.Title,
		"contacts": []map[string]string{
			{"id": params.ContactID},
		},
	}
	if params.Description != "" {
		body["description"] = params.Description
	}
	if len(params.Attributes) > 0 {
		attrs := make(map[string]string, len(params.Attributes))
		for _, a := range params.Attributes {
			attrs[a.Name] = a.Value
		}
		body["ticket_attributes"] = attrs
	}

	var resp intercomTicket
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tickets", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
