package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listCommitsAction implements connectors.Action for github.list_commits.
// It lists commits via GET /repos/{owner}/{repo}/commits.
type listCommitsAction struct {
	conn *GitHubConnector
}

type listCommitsParams struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	SHA     string `json:"sha"`
	Path    string `json:"path"`
	Author  string `json:"author"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

func (p *listCommitsParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.Path != "" {
		if err := validateFilePath(p.Path); err != nil {
			return err
		}
	}
	return validatePerPage(p.PerPage)
}

// Execute lists commits for a repository.
func (a *listCommitsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listCommitsParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.SHA != "" {
		q.Set("sha", params.SHA)
	}
	if params.Path != "" {
		q.Set("path", params.Path)
	}
	if params.Author != "" {
		q.Set("author", params.Author)
	}
	setPagination(q, params.PerPage, params.Page)

	apiPath := fmt.Sprintf("/repos/%s/%s/commits?%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), q.Encode())

	var ghResp []map[string]any
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiPath, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
