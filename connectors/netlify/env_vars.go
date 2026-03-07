package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listEnvVarsAction implements netlify.list_env_vars.
type listEnvVarsAction struct {
	conn *NetlifyConnector
}

type listEnvVarsParams struct {
	AccountSlug string `json:"account_slug"`
	SiteID      string `json:"site_id"`
}

func (p *listEnvVarsParams) validate() error {
	if p.AccountSlug == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_slug"}
	}
	if p.SiteID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: site_id"}
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

	path := "/accounts/" + url.PathEscape(params.AccountSlug) + "/env"
	path += "?site_id=" + url.QueryEscape(params.SiteID)

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}

// setEnvVarAction implements netlify.set_env_var.
type setEnvVarAction struct {
	conn *NetlifyConnector
}

type envVarValue struct {
	Value   string `json:"value"`
	Context string `json:"context"`
}

type setEnvVarParams struct {
	AccountSlug string        `json:"account_slug"`
	SiteID      string        `json:"site_id"`
	Key         string        `json:"key"`
	Values      []envVarValue `json:"values"`
}

func (p *setEnvVarParams) validate() error {
	if p.AccountSlug == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_slug"}
	}
	if p.SiteID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: site_id"}
	}
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	if len(p.Values) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: values"}
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

	body := []map[string]interface{}{
		{
			"key":    params.Key,
			"scopes": []string{"builds", "functions", "runtime", "post_processing"},
			"values": params.Values,
		},
	}

	path := "/accounts/" + url.PathEscape(params.AccountSlug) + "/env"
	path += "?site_id=" + url.QueryEscape(params.SiteID)

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}

// deleteEnvVarAction implements netlify.delete_env_var.
type deleteEnvVarAction struct {
	conn *NetlifyConnector
}

type deleteEnvVarParams struct {
	AccountSlug string `json:"account_slug"`
	Key         string `json:"key"`
	SiteID      string `json:"site_id"`
}

func (p *deleteEnvVarParams) validate() error {
	if p.AccountSlug == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_slug"}
	}
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
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

	path := "/accounts/" + url.PathEscape(params.AccountSlug) + "/env/" + url.PathEscape(params.Key)
	if params.SiteID != "" {
		path += "?site_id=" + url.QueryEscape(params.SiteID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, &resp); err != nil {
		return nil, err
	}

	if resp == nil || string(resp) == "null" {
		return connectors.JSONResult(map[string]string{
			"status": "deleted",
			"key":    params.Key,
		})
	}
	return connectors.JSONResult(resp)
}
