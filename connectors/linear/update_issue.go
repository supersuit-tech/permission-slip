package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateIssueAction implements connectors.Action for linear.update_issue.
type updateIssueAction struct {
	conn *LinearConnector
}

type updateIssueParams struct {
	IssueID     string   `json:"issue_id"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	AssigneeID  string   `json:"assignee_id,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	StateID     string   `json:"state_id,omitempty"`
	LabelIDs    []string `json:"label_ids,omitempty"`
}

func (p *updateIssueParams) validate() error {
	if p.IssueID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_id"}
	}
	if p.Priority != nil && (*p.Priority < 0 || *p.Priority > 4) {
		return &connectors.ValidationError{Message: "priority must be 0 (none), 1 (urgent), 2 (high), 3 (medium), or 4 (low)"}
	}
	if !p.hasUpdates() {
		return &connectors.ValidationError{Message: "at least one field to update must be provided (title, description, assignee_id, priority, state_id, or label_ids)"}
	}
	return nil
}

func (p *updateIssueParams) hasUpdates() bool {
	return p.Title != "" || p.Description != "" || p.AssigneeID != "" ||
		p.Priority != nil || p.StateID != "" || len(p.LabelIDs) > 0
}

const updateIssueMutation = `mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
	issueUpdate(id: $id, input: $input) {
		success
		issue {
			id
			identifier
			title
			url
			state {
				name
			}
		}
	}
}`

type updateIssueResponse struct {
	IssueUpdate struct {
		Success bool `json:"success"`
		Issue   struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
			URL        string `json:"url"`
			State      struct {
				Name string `json:"name"`
			} `json:"state"`
		} `json:"issue"`
	} `json:"issueUpdate"`
}

func (a *updateIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	input := map[string]any{}
	if params.Title != "" {
		input["title"] = params.Title
	}
	if params.Description != "" {
		input["description"] = params.Description
	}
	if params.AssigneeID != "" {
		input["assigneeId"] = params.AssigneeID
	}
	if params.Priority != nil {
		input["priority"] = *params.Priority
	}
	if params.StateID != "" {
		input["stateId"] = params.StateID
	}
	if len(params.LabelIDs) > 0 {
		input["labelIds"] = params.LabelIDs
	}

	vars := map[string]any{
		"id":    params.IssueID,
		"input": input,
	}

	var resp updateIssueResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, updateIssueMutation, vars, &resp); err != nil {
		return nil, err
	}

	if !resp.IssueUpdate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear issueUpdate returned success=false"}
	}

	return connectors.JSONResult(map[string]string{
		"id":         resp.IssueUpdate.Issue.ID,
		"identifier": resp.IssueUpdate.Issue.Identifier,
		"title":      resp.IssueUpdate.Issue.Title,
		"url":        resp.IssueUpdate.Issue.URL,
		"state":      resp.IssueUpdate.Issue.State.Name,
	})
}
