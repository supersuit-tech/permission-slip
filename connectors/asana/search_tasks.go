package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type searchTasksAction struct {
	conn *AsanaConnector
}

type searchTasksParams struct {
	WorkspaceID string   `json:"workspace_id"`
	Text        string   `json:"text"`
	Assignee    string   `json:"assignee"`
	Projects    []string `json:"projects"`
	Completed   *bool    `json:"completed"`
	DueOnBefore string   `json:"due_on_before"`
	DueOnAfter  string   `json:"due_on_after"`
	Limit       int      `json:"limit"`
}

func (p *searchTasksParams) validate() error {
	if p.WorkspaceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: workspace_id"}
	}
	return nil
}

func (a *searchTasksAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchTasksParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	// Fall back to workspace_id from credentials if not provided in parameters.
	if params.WorkspaceID == "" {
		if wsID, ok := req.Credentials.Get("workspace_id"); ok && wsID != "" {
			params.WorkspaceID = wsID
		}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.Text != "" {
		q.Set("text", params.Text)
	}
	if params.Assignee != "" {
		q.Set("assignee.any", params.Assignee)
	}
	for _, p := range params.Projects {
		q.Add("projects.any", p)
	}
	if params.Completed != nil {
		q.Set("completed", strconv.FormatBool(*params.Completed))
	}
	if params.DueOnBefore != "" {
		q.Set("due_on.before", params.DueOnBefore)
	}
	if params.DueOnAfter != "" {
		q.Set("due_on.after", params.DueOnAfter)
	}
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	q.Set("limit", strconv.Itoa(limit))

	fullURL := fmt.Sprintf("%s/workspaces/%s/tasks/search?%s", a.conn.baseURL, url.PathEscape(params.WorkspaceID), q.Encode())

	// Search returns {"data": [...]} — use doRaw to avoid the request body envelope.
	var envelope struct {
		Data []struct {
			GID          string `json:"gid"`
			Name         string `json:"name"`
			Completed    bool   `json:"completed"`
			PermalinkURL string `json:"permalink_url"`
		} `json:"data"`
	}

	if err := a.conn.doRaw(ctx, req.Credentials, http.MethodGet, fullURL, &envelope); err != nil {
		return nil, err
	}

	return connectors.JSONResult(envelope.Data)
}
