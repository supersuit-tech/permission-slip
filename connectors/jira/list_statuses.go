package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listStatusesAction implements connectors.Action for jira.list_statuses.
// It lists available statuses via GET /rest/api/3/status. Optionally filters
// by project key.
type listStatusesAction struct {
	conn *JiraConnector
}

type listStatusesParams struct {
	ProjectKey string `json:"project_key"`
}

func (a *listStatusesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listStatusesParams
	if req.Parameters != nil {
		// Parameters are optional for this action, but invalid JSON should be reported.
		if err := json.Unmarshal(req.Parameters, &params); err != nil {
			return nil, &connectors.ValidationError{Message: "invalid parameters JSON: " + err.Error()}
		}
	}

	path := "/status"
	if params.ProjectKey != "" {
		path += "?projectKey=" + url.QueryEscape(params.ProjectKey)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
