package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listProjectsAction struct {
	conn *AsanaConnector
}

type listProjectsParams struct {
	WorkspaceID string `json:"workspace_id"`
}

func (p *listProjectsParams) validate() error {
	if p.WorkspaceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: workspace_id"}
	}
	return nil
}

func (a *listProjectsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listProjectsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	// Fall back to workspace_id from credentials if not provided in parameters.
	if params.WorkspaceID == "" {
		if wsID, ok := req.Credentials.Get("workspace_id"); ok && wsID != "" {
			params.WorkspaceID = wsID
		}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("workspace", params.WorkspaceID)

	fullURL := fmt.Sprintf("%s/projects?%s", a.conn.baseURL, q.Encode())

	var envelope struct {
		Data []struct {
			GID          string `json:"gid"`
			Name         string `json:"name"`
			PermalinkURL string `json:"permalink_url"`
		} `json:"data"`
	}

	if err := a.conn.doRaw(ctx, req.Credentials, "GET", fullURL, &envelope); err != nil {
		return nil, err
	}

	return connectors.JSONResult(envelope.Data)
}
