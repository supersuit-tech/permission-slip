package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// moveToSprintAction implements connectors.Action for jira.move_to_sprint.
// It moves issues to a sprint via POST /rest/agile/1.0/sprint/{sprintId}/issue.
type moveToSprintAction struct {
	conn *JiraConnector
}

type moveToSprintParams struct {
	SprintID int      `json:"sprint_id"`
	Issues   []string `json:"issues"`
}

func (p *moveToSprintParams) validate() error {
	if p.SprintID <= 0 {
		return &connectors.ValidationError{Message: "missing required parameter: sprint_id (must be a positive integer)"}
	}
	if len(p.Issues) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: issues (at least one issue key required)"}
	}
	return nil
}

func (a *moveToSprintAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params moveToSprintParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"issues": params.Issues,
	}

	path := "/sprint/" + strconv.Itoa(params.SprintID) + "/issue"
	if err := a.conn.doAgile(ctx, req.Credentials, http.MethodPost, path, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]interface{}{
		"sprint_id": params.SprintID,
		"issues":    params.Issues,
		"status":    "moved",
	})
}
