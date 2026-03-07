package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listTagsAction implements connectors.Action for zendesk.list_tags.
// It retrieves tags for a ticket via GET /tickets/{id}/tags.json.
type listTagsAction struct {
	conn *ZendeskConnector
}

type listTagsParams struct {
	TicketID int64 `json:"ticket_id"`
}

type tagsResponse struct {
	Tags []string `json:"tags"`
}

func (p *listTagsParams) validate() error {
	if !isValidZendeskID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	return nil
}

func (a *listTagsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listTagsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp tagsResponse
	path := fmt.Sprintf("/tickets/%d/tags.json", params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
