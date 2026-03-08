package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getRepoAction implements connectors.Action for github.get_repo.
// It fetches repository metadata via GET /repos/{owner}/{repo}.
type getRepoAction struct {
	conn *GitHubConnector
}

type getRepoParams struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

func (p *getRepoParams) validate() error {
	return requireOwnerRepo(p.Owner, p.Repo)
}

// Execute retrieves metadata for a GitHub repository.
func (a *getRepoAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[getRepoParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	var ghResp struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Private       bool   `json:"private"`
		HTMLURL       string `json:"html_url"`
		Description   string `json:"description"`
		Fork          bool   `json:"fork"`
		DefaultBranch string `json:"default_branch"`
		OpenIssues    int    `json:"open_issues_count"`
		StargazersCount int  `json:"stargazers_count"`
	}

	path := fmt.Sprintf("/repos/%s/%s", url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
