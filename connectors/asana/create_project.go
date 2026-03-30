package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createProjectAction struct {
	conn *AsanaConnector
}

type createProjectParams struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Notes       string `json:"notes"`
	Color       string `json:"color"`
	Privacy     string `json:"privacy"`
}

func (p *createProjectParams) validate() error {
	if p.WorkspaceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: workspace_id"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (a *createProjectAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createProjectParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"workspace": params.WorkspaceID,
		"name":      params.Name,
	}
	if params.Notes != "" {
		body["notes"] = params.Notes
	}
	if params.Color != "" {
		body["color"] = params.Color
	}
	if params.Privacy != "" {
		body["privacy_setting"] = params.Privacy
	}

	var resp struct {
		GID          string `json:"gid"`
		Name         string `json:"name"`
		PermalinkURL string `json:"permalink_url"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/projects", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
