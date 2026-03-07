package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listDeploymentsAction struct {
	conn *VercelConnector
}

type listDeploymentsParams struct {
	ProjectID string `json:"project_id"`
	TeamID    string `json:"team_id"`
	Target    string `json:"target"`
	State     string `json:"state"`
	Limit     int    `json:"limit"`
}

func (a *listDeploymentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDeploymentsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	q := url.Values{}
	if params.ProjectID != "" {
		q.Set("projectId", params.ProjectID)
	}
	if params.TeamID != "" {
		q.Set("teamId", params.TeamID)
	}
	if params.Target != "" {
		q.Set("target", params.Target)
	}
	if params.State != "" {
		q.Set("state", params.State)
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}

	path := "/v6/deployments"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
