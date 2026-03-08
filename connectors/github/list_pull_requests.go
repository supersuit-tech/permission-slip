package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listPullRequestsAction implements connectors.Action for github.list_pull_requests.
// It lists pull requests via GET /repos/{owner}/{repo}/pulls.
type listPullRequestsAction struct {
	conn *GitHubConnector
}

type listPullRequestsParams struct {
	Owner     string `json:"owner"`
	Repo      string `json:"repo"`
	State     string `json:"state"`
	Base      string `json:"base"`
	Head      string `json:"head"`
	Sort      string `json:"sort"`
	Direction string `json:"direction"`
	PerPage   int    `json:"per_page"`
	Page      int    `json:"page"`
}

func (p *listPullRequestsParams) validate() error {
	return requireOwnerRepo(p.Owner, p.Repo)
}

// Execute lists pull requests for a GitHub repository.
func (a *listPullRequestsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listPullRequestsParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if params.State != "" {
		query.Set("state", params.State)
	}
	if params.Base != "" {
		query.Set("base", params.Base)
	}
	if params.Head != "" {
		query.Set("head", params.Head)
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Direction != "" {
		query.Set("direction", params.Direction)
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	if params.Page > 1 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls?%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), query.Encode())

	var ghResp []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		Draft   bool   `json:"draft"`
		Body    string `json:"body"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
