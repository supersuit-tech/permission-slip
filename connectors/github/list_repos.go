package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listReposAction implements connectors.Action for github.list_repos.
// It lists repositories for the authenticated user (GET /user/repos) or
// for an organization (GET /orgs/{org}/repos).
type listReposAction struct {
	conn *GitHubConnector
}

type listReposParams struct {
	Org      string `json:"org"`
	Type     string `json:"type"`
	Sort     string `json:"sort"`
	PerPage  int    `json:"per_page"`
}

func (p *listReposParams) validate() error {
	return nil
}

// Execute lists repositories for the authenticated user or an organization.
func (a *listReposAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listReposParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	var basePath string
	if params.Org != "" {
		basePath = fmt.Sprintf("/orgs/%s/repos", url.PathEscape(params.Org))
	} else {
		basePath = "/user/repos"
	}

	query := url.Values{}
	if params.Type != "" {
		query.Set("type", params.Type)
	}
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))

	path := basePath + "?" + query.Encode()

	var ghResp []struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
		HTMLURL  string `json:"html_url"`
		Fork     bool   `json:"fork"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
