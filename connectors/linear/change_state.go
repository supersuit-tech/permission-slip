package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// changeStateAction implements connectors.Action for linear.change_state.
type changeStateAction struct {
	conn *LinearConnector
}

type changeStateParams struct {
	IssueID string `json:"issue_id"`
	StateID string `json:"state_id"`
}

func (p *changeStateParams) validate() error {
	if p.IssueID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_id"}
	}
	if p.StateID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: state_id"}
	}
	return nil
}

const changeStateMutation = `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
	issueUpdate(id: $id, input: $input) {
		success
		issue {
			id
			identifier
			state {
				id
				name
			}
		}
	}
}`

type changeStateResponse struct {
	IssueUpdate struct {
		Success bool `json:"success"`
		Issue   struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			State      struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"state"`
		} `json:"issue"`
	} `json:"issueUpdate"`
}

func (a *changeStateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params changeStateParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	vars := map[string]any{
		"id":    params.IssueID,
		"input": map[string]any{"stateId": params.StateID},
	}

	var resp changeStateResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, changeStateMutation, vars, &resp); err != nil {
		return nil, err
	}

	if !resp.IssueUpdate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear issueUpdate returned success=false"}
	}

	return connectors.JSONResult(map[string]string{
		"id":         resp.IssueUpdate.Issue.ID,
		"identifier": resp.IssueUpdate.Issue.Identifier,
		"state_id":   resp.IssueUpdate.Issue.State.ID,
		"state_name": resp.IssueUpdate.Issue.State.Name,
	})
}
