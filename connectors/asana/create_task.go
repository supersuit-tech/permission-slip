package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createTaskAction struct {
	conn *AsanaConnector
}

type createTaskParams struct {
	ProjectID    string         `json:"project_id"`
	Name         string         `json:"name"`
	Notes        string         `json:"notes"`
	Assignee     string         `json:"assignee"`
	DueOn        string         `json:"due_on"`
	DueAt        string         `json:"due_at"`
	Tags         []string       `json:"tags"`
	CustomFields map[string]any `json:"custom_fields"`
}

func (p *createTaskParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (a *createTaskAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTaskParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name":     params.Name,
		"projects": []string{params.ProjectID},
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
	if len(params.Tags) > 0 {
		body["tags"] = params.Tags
	}
	if len(params.CustomFields) > 0 {
		body["custom_fields"] = params.CustomFields
	}

	var resp struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		PermalinkURL string `json:"permalink_url"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/tasks", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
