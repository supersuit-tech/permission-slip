package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createSectionAction struct {
	conn *AsanaConnector
}

type createSectionParams struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

func (p *createSectionParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (a *createSectionAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSectionParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"name": params.Name,
	}

	path := fmt.Sprintf("/projects/%s/sections", url.PathEscape(params.ProjectID))

	var resp struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
