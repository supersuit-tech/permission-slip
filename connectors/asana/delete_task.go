package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type deleteTaskAction struct {
	conn *AsanaConnector
}

type deleteTaskParams struct {
	TaskID string `json:"task_id"`
}

func (p *deleteTaskParams) validate() error {
	if p.TaskID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: task_id"}
	}
	return nil
}

func (a *deleteTaskAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteTaskParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/tasks/%s", url.PathEscape(params.TaskID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"task_id": params.TaskID,
		"status":  "deleted",
	})
}
