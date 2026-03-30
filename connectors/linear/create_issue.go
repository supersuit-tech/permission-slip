package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createIssueAction implements connectors.Action for linear.create_issue.
type createIssueAction struct {
	conn *LinearConnector
}

type createIssueParams struct {
	TeamID      string   `json:"team_id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	AssigneeID  string   `json:"assignee_id,omitempty"`
	Priority    *int     `json:"priority,omitempty"`
	StateID     string   `json:"state_id,omitempty"`
	LabelIDs    []string `json:"label_ids,omitempty"`
	ProjectID   string   `json:"project_id,omitempty"`
}

func (p *createIssueParams) validate() error {
	if p.TeamID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: team_id"}
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if err := validatePriority(p.Priority); err != nil {
		return err
	}
	return nil
}

const createIssueMutation = `mutation IssueCreate($input: IssueCreateInput!) {
	issueCreate(input: $input) {
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

type createIssueResponse struct {
	IssueCreate struct {
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
	} `json:"issueCreate"`
}

func (a *createIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	input := map[string]any{
		"teamId": params.TeamID,
		"title":  params.Title,
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
	if params.ProjectID != "" {
		input["projectId"] = params.ProjectID
	}

	var resp createIssueResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, createIssueMutation, map[string]any{"input": input}, &resp); err != nil {
		return nil, err
	}

	if !resp.IssueCreate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear issueCreate returned success=false"}
	}

	return connectors.JSONResult(map[string]string{
		"id":         resp.IssueCreate.Issue.ID,
		"identifier": resp.IssueCreate.Issue.Identifier,
		"title":      resp.IssueCreate.Issue.Title,
		"url":        resp.IssueCreate.Issue.URL,
		"state":      resp.IssueCreate.Issue.State.Name,
	})
}
