package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listWorkspacesAction struct {
	conn *AsanaConnector
}

type listWorkspacesParams struct{}

func (a *listWorkspacesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listWorkspacesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	var resp []struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/workspaces", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
