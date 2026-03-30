package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// assignIssueAction implements connectors.Action for linear.assign_issue.
type assignIssueAction struct {
	conn *LinearConnector
}

type assignIssueParams struct {
	IssueID    string `json:"issue_id"`
	AssigneeID string `json:"assignee_id"`
}

func (p *assignIssueParams) validate() error {
	if p.IssueID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_id"}
	}
	// assignee_id may be empty to unassign.
	return nil
}

const assignIssueMutation = `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
	issueUpdate(id: $id, input: $input) {
		success
		issue {
			id
			identifier
			assignee {
				id
				name
			}
		}
	}
}`

type assignIssueResponse struct {
	IssueUpdate struct {
		Success bool `json:"success"`
		Issue   struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Assignee   *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"assignee"`
		} `json:"issue"`
	} `json:"issueUpdate"`
}

func (a *assignIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params assignIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	input := map[string]any{}
	if params.AssigneeID != "" {
		input["assigneeId"] = params.AssigneeID
	} else {
		input["assigneeId"] = nil
	}

	vars := map[string]any{
		"id":    params.IssueID,
		"input": input,
	}

	var resp assignIssueResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, assignIssueMutation, vars, &resp); err != nil {
		return nil, err
	}

	if !resp.IssueUpdate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear issueUpdate returned success=false"}
	}

	result := map[string]string{
		"id":         resp.IssueUpdate.Issue.ID,
		"identifier": resp.IssueUpdate.Issue.Identifier,
	}
	if resp.IssueUpdate.Issue.Assignee != nil {
		result["assignee_id"] = resp.IssueUpdate.Issue.Assignee.ID
		result["assignee_name"] = resp.IssueUpdate.Issue.Assignee.Name
	}

	return connectors.JSONResult(result)
}
