package jira

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listProjectsAction implements connectors.Action for jira.list_projects.
// It lists projects via GET /rest/api/3/project.
type listProjectsAction struct {
	conn *JiraConnector
}

func (a *listProjectsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/project", nil, &resp); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: resp}, nil
}
