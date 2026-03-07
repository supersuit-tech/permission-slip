package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateTagsAction implements connectors.Action for zendesk.update_tags.
// It replaces a ticket's tags via PUT /tickets/{id}/tags.json.
type updateTagsAction struct {
	conn *ZendeskConnector
}

type updateTagsParams struct {
	TicketID int64    `json:"ticket_id"`
	Tags     []string `json:"tags"`
}

// maxTags limits the number of tags per update to prevent abuse.
const maxTags = 100

func (p *updateTagsParams) validate() error {
	if !isValidZendeskID(p.TicketID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: ticket_id"}
	}
	if len(p.Tags) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: tags"}
	}
	if len(p.Tags) > maxTags {
		return &connectors.ValidationError{Message: fmt.Sprintf("tags cannot exceed %d items", maxTags)}
	}
	return nil
}

func (a *updateTagsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateTagsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string][]string{"tags": params.Tags}
	var resp tagsResponse
	path := fmt.Sprintf("/tickets/%d/tags.json", params.TicketID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
