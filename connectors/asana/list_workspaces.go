package asana

import (
	"context"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listWorkspacesAction struct {
	conn *AsanaConnector
}

func (a *listWorkspacesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp []struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/workspaces", nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
