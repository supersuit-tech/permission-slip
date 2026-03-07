package zendesk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// mergeTicketsAction implements connectors.Action for zendesk.merge_tickets.
// It merges source tickets into a target via POST /tickets/{id}/merge.json.
// This is a destructive, irreversible operation — risk level: high.
type mergeTicketsAction struct {
	conn *ZendeskConnector
}

type mergeTicketsParams struct {
	TargetID  int64   `json:"target_id"`
	SourceIDs []int64 `json:"source_ids"`
}

// maxMergeSourceIDs limits the number of source tickets per merge request.
const maxMergeSourceIDs = 5

func (p *mergeTicketsParams) validate() error {
	if !isValidZendeskID(p.TargetID) {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: target_id"}
	}
	if len(p.SourceIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: source_ids"}
	}
	if len(p.SourceIDs) > maxMergeSourceIDs {
		return &connectors.ValidationError{Message: fmt.Sprintf("source_ids cannot exceed %d tickets", maxMergeSourceIDs)}
	}
	for i, id := range p.SourceIDs {
		if !isValidZendeskID(id) {
			return &connectors.ValidationError{Message: fmt.Sprintf("source_ids[%d] must be a positive integer", i)}
		}
		if id == p.TargetID {
			return &connectors.ValidationError{Message: fmt.Sprintf("source_ids[%d] cannot be the same as target_id", i)}
		}
	}
	return nil
}

func (a *mergeTicketsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params mergeTicketsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"ids":            params.SourceIDs,
		"target_comment": "Merged via Permission Slip",
		"source_comment": "Merged into another ticket via Permission Slip",
	}

	var resp jobStatusResponse
	path := fmt.Sprintf("/tickets/%d/merge.json", params.TargetID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
