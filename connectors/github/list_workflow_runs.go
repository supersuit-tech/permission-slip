package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listWorkflowRunsAction implements connectors.Action for github.list_workflow_runs.
// It lists workflow runs via GET /repos/{owner}/{repo}/actions/runs or
// GET /repos/{owner}/{repo}/actions/workflows/{workflow_id}/runs when a
// workflow_id is provided.
type listWorkflowRunsAction struct {
	conn *GitHubConnector
}

type listWorkflowRunsParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
	Branch     string `json:"branch"`
	Event      string `json:"event"`
	Actor      string `json:"actor"`
	PerPage    int    `json:"per_page"`
	Page       int    `json:"page"`
}

func (p *listWorkflowRunsParams) validate() error {
	return requireOwnerRepo(p.Owner, p.Repo)
}

// Execute lists workflow runs for a GitHub repository.
func (a *listWorkflowRunsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listWorkflowRunsParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	var basePath string
	if params.WorkflowID != "" {
		basePath = fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/runs",
			url.PathEscape(params.Owner), url.PathEscape(params.Repo), url.PathEscape(params.WorkflowID))
	} else {
		basePath = fmt.Sprintf("/repos/%s/%s/actions/runs",
			url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	}

	query := url.Values{}
	if params.Status != "" {
		query.Set("status", params.Status)
	}
	if params.Branch != "" {
		query.Set("branch", params.Branch)
	}
	if params.Event != "" {
		query.Set("event", params.Event)
	}
	if params.Actor != "" {
		query.Set("actor", params.Actor)
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	if params.Page > 1 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}

	path := basePath + "?" + query.Encode()

	var ghResp struct {
		TotalCount   int `json:"total_count"`
		WorkflowRuns []struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Event      string `json:"event"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			HTMLURL    string `json:"html_url"`
			HeadBranch string `json:"head_branch"`
			HeadSHA    string `json:"head_sha"`
			CreatedAt  string `json:"created_at"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"workflow_runs"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
