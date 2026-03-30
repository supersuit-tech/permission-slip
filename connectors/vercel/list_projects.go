package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listProjectsAction struct {
	conn *VercelConnector
}

type listProjectsParams struct {
	TeamID string `json:"team_id"`
	Limit  int    `json:"limit"`
}

func (a *listProjectsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listProjectsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	q := url.Values{}
	if params.TeamID != "" {
		q.Set("teamId", params.TeamID)
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}

	path := "/v9/projects"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
