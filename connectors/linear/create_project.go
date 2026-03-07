package linear

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createProjectAction implements connectors.Action for linear.create_project.
type createProjectAction struct {
	conn *LinearConnector
}

type createProjectParams struct {
	TeamIDs     []string `json:"team_ids"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	State       string   `json:"state,omitempty"`
}

var validProjectStates = map[string]bool{
	"planned":   true,
	"started":   true,
	"paused":    true,
	"completed": true,
	"cancelled": true,
}

func (p *createProjectParams) validate() error {
	if len(p.TeamIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: team_ids"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.State != "" && !validProjectStates[p.State] {
		return &connectors.ValidationError{Message: "state must be one of: planned, started, paused, completed, cancelled"}
	}
	return nil
}

const createProjectMutation = `mutation ProjectCreate($input: ProjectCreateInput!) {
	projectCreate(input: $input) {
		success
		project {
			id
			name
			url
		}
	}
}`

type createProjectResponse struct {
	ProjectCreate struct {
		Success bool `json:"success"`
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"project"`
	} `json:"projectCreate"`
}

func (a *createProjectAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createProjectParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	input := map[string]any{
		"teamIds": params.TeamIDs,
		"name":    params.Name,
	}
	if params.Description != "" {
		input["description"] = params.Description
	}
	if params.State != "" {
		input["state"] = params.State
	}

	var resp createProjectResponse
	if err := a.conn.doGraphQL(ctx, req.Credentials, createProjectMutation, map[string]any{"input": input}, &resp); err != nil {
		return nil, err
	}

	if !resp.ProjectCreate.Success {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Linear projectCreate returned success=false"}
	}

	return connectors.JSONResult(map[string]string{
		"id":   resp.ProjectCreate.Project.ID,
		"name": resp.ProjectCreate.Project.Name,
		"url":  resp.ProjectCreate.Project.URL,
	})
}
