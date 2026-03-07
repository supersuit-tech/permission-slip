package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createSubtaskAction struct {
	conn *AsanaConnector
}

type createSubtaskParams struct {
	ParentTaskID string `json:"parent_task_id"`
	Name         string `json:"name"`
	Notes        string `json:"notes"`
	Assignee     string `json:"assignee"`
	DueOn        string `json:"due_on"`
}

func (p *createSubtaskParams) validate() error {
	if p.ParentTaskID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: parent_task_id"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (a *createSubtaskAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSubtaskParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name": params.Name,
	}
	if params.Notes != "" {
		body["notes"] = params.Notes
	}
	if params.Assignee != "" {
		body["assignee"] = params.Assignee
	}
	if params.DueOn != "" {
		body["due_on"] = params.DueOn
	}

	var resp struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		PermalinkURL string `json:"permalink_url"`
	}

	path := fmt.Sprintf("/tasks/%s/subtasks", url.PathEscape(params.ParentTaskID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
