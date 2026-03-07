package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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

	path := "/v9/projects"
	sep := "?"
	if params.TeamID != "" {
		path += sep + "teamId=" + params.TeamID
		sep = "&"
	}
	if params.Limit > 0 {
		path += fmt.Sprintf("%slimit=%d", sep, params.Limit)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return connectors.JSONResult(resp)
}
