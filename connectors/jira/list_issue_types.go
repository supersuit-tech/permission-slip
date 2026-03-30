package jira

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listIssueTypesAction implements connectors.Action for jira.list_issue_types.
// It lists available issue types via GET /rest/api/3/issuetype.
type listIssueTypesAction struct {
	conn *JiraConnector
}

func (a *listIssueTypesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/issuetype", nil, &resp); err != nil {
		return nil, err
	}
	return &connectors.ActionResult{Data: resp}, nil
}
