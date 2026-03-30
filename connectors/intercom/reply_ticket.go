package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// replyTicketAction implements connectors.Action for intercom.reply_ticket.
// It adds a reply to a ticket via POST /tickets/{id}/reply.
type replyTicketAction struct {
	conn *IntercomConnector
}

type replyTicketParams struct {
	TicketID    string `json:"ticket_id"`
	Body        string `json:"body"`
	MessageType string `json:"message_type"` // "comment" (public) or "note" (internal)
	AdminID     string `json:"admin_id"`
}

var validMessageTypes = map[string]bool{
	"comment": true, "note": true,
}

func (p *replyTicketParams) validate() error {
	if !isValidIntercomID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if p.AdminID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: admin_id"}
	}
	if p.MessageType == "" {
		p.MessageType = "comment"
	}
	if !validMessageTypes[p.MessageType] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid message_type %q: must be comment or note", p.MessageType)}
	}
	return nil
}

func (a *replyTicketAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params replyTicketParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"body":         params.Body,
		"message_type": params.MessageType,
		"type":         "admin",
		"admin_id":     params.AdminID,
	}

	var resp intercomTicket
	path := "/tickets/" + url.PathEscape(params.TicketID) + "/reply"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
