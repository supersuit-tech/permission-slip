package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type updateTaskAction struct {
	conn *AsanaConnector
}

type updateTaskParams struct {
	TaskID       string         `json:"task_id"`
	Name         string         `json:"name"`
	Notes        string         `json:"notes"`
	Assignee     string         `json:"assignee"`
	DueOn        string         `json:"due_on"`
	DueAt        string         `json:"due_at"`
	Completed    *bool          `json:"completed"`
	CustomFields map[string]any `json:"custom_fields"`
}

func (p *updateTaskParams) validate() error {
	if p.TaskID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: task_id"}
	}
	return nil
}

func (a *updateTaskAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateTaskParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.Name != "" {
		body["name"] = params.Name
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
	if params.DueAt != "" {
		body["due_at"] = params.DueAt
	}
	if params.Completed != nil {
		body["completed"] = *params.Completed
	}
	if len(params.CustomFields) > 0 {
		body["custom_fields"] = params.CustomFields
	}

	var resp struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		PermalinkURL string `json:"permalink_url"`
	}

	path := fmt.Sprintf("/tasks/%s", url.PathEscape(params.TaskID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
