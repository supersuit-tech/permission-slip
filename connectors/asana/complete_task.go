package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type completeTaskAction struct {
	conn *AsanaConnector
}

type completeTaskParams struct {
	TaskID string `json:"task_id"`
}

func (p *completeTaskParams) validate() error {
	if p.TaskID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: task_id"}
	}
	return nil
}

func (a *completeTaskAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params completeTaskParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"completed": true,
	}

	var resp struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		Completed    bool   `json:"completed"`
		PermalinkURL string `json:"permalink_url"`
	}

	path := fmt.Sprintf("/tasks/%s", url.PathEscape(params.TaskID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
