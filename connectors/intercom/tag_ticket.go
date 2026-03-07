package intercom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// tagTicketAction implements connectors.Action for intercom.tag_ticket.
// It applies a tag to a ticket via POST /tags.
type tagTicketAction struct {
	conn *IntercomConnector
}

type tagTicketParams struct {
	TagName  string `json:"tag_name"`
	TicketID string `json:"ticket_id"`
}

func (p *tagTicketParams) validate() error {
	if p.TagName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tag_name"}
	}
	if p.TicketID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ticket_id"}
	}
	return nil
}

func (a *tagTicketAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params tagTicketParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name": params.TagName,
		"tickets": []map[string]string{
			{"id": params.TicketID},
		},
	}

	var resp tag
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tags", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
