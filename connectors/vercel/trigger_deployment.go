package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type triggerDeploymentAction struct {
	conn *VercelConnector
}

type triggerDeploymentParams struct {
	ProjectID string `json:"project_id"`
	Ref       string `json:"ref"`
	RefType   string `json:"ref_type"`
	Target    string `json:"target"`
	TeamID    string `json:"team_id"`
}

func (p *triggerDeploymentParams) validate() error {
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if p.Ref == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ref"}
	}
	return nil
}

func (a *triggerDeploymentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params triggerDeploymentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	target := params.Target
	if target == "" {
		target = "preview"
	}

	refType := params.RefType
	if refType == "" {
		refType = "branch"
	}

	body := map[string]interface{}{
		"name":   params.ProjectID,
		"target": target,
		"gitSource": map[string]string{
			"ref":  params.Ref,
			"type": refType,
		},
	}

	path := "/v13/deployments"
	if params.TeamID != "" {
		path += "?teamId=" + url.QueryEscape(params.TeamID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
