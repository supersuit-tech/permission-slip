package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// replyTicketAction implements connectors.Action for zendesk.reply_ticket.
// It adds a comment to a ticket via PUT /tickets/{id}.json.
type replyTicketAction struct {
	conn *ZendeskConnector
}

type replyTicketParams struct {
	TicketID int64  `json:"ticket_id"`
	Body     string `json:"body"`
	Public   bool   `json:"public"`
}

func (p *replyTicketParams) validate() error {
	if !isValidZendeskID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
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
		"ticket": map[string]any{
			"comment": ticketComment{
				Body:   params.Body,
				Public: params.Public,
			},
		},
	}

	var resp ticketResponse
	path := fmt.Sprintf("/tickets/%d.json", params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
