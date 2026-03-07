package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listEnvVarsAction implements vercel.list_env_vars.
type listEnvVarsAction struct {
	conn *VercelConnector
}

type listEnvVarsParams struct {
	ProjectID string `json:"project_id"`
	TeamID    string `json:"team_id"`
}

func (p *listEnvVarsParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	return nil
}

func (a *listEnvVarsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listEnvVarsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/v9/projects/" + url.PathEscape(params.ProjectID) + "/env"
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}

// setEnvVarAction implements vercel.set_env_var.
type setEnvVarAction struct {
	conn *VercelConnector
}

type setEnvVarParams struct {
	ProjectID string   `json:"project_id"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	Target    []string `json:"target"`
	Type      string   `json:"type"`
	TeamID    string   `json:"team_id"`
}

func (p *setEnvVarParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	if p.Value == "" {
		return &connectors.ValidationError{Message: "missing required parameter: value"}
	}
	if len(p.Target) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: target"}
	}
	return nil
}

func (a *setEnvVarAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params setEnvVarParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	envType := params.Type
	if envType == "" {
		envType = "encrypted"
	}

	body := map[string]interface{}{
		"key":    params.Key,
		"value":  params.Value,
		"target": params.Target,
		"type":   envType,
	}

	path := "/v10/projects/" + url.PathEscape(params.ProjectID) + "/env"
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}

// deleteEnvVarAction implements vercel.delete_env_var.
type deleteEnvVarAction struct {
	conn *VercelConnector
}

type deleteEnvVarParams struct {
	ProjectID string `json:"project_id"`
	EnvID     string `json:"env_id"`
	TeamID    string `json:"team_id"`
}

func (p *deleteEnvVarParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.EnvID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: env_id"}
	}
	return nil
}

func (a *deleteEnvVarAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteEnvVarParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/v9/projects/" + url.PathEscape(params.ProjectID) + "/env/" + url.PathEscape(params.EnvID)
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &resp); err != nil {
		return nil, err
	}

	if resp == nil || string(resp) == "null" {
		return connectors.JSONResult(map[string]string{
			"status": "deleted",
			"env_id": params.EnvID,
		})
	}
	return connectors.JSONResult(resp)
}
