package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchIssuesAction implements connectors.Action for github.search_issues.
// It searches issues and pull requests across repositories via GET /search/issues.
type searchIssuesAction struct {
	conn *GitHubConnector
}

type searchIssuesParams struct {
	Q       string `json:"q"`
	Sort    string `json:"sort"`
	Order   string `json:"order"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

func (p *searchIssuesParams) validate() error {
	if p.Q == "" {
		return &connectors.ValidationError{Message: "missing required parameter: q"}
	}
	return nil
}

// Execute searches issues and pull requests across GitHub repositories.
func (a *searchIssuesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[searchIssuesParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("q", params.Q)
	if params.Sort != "" {
		query.Set("sort", params.Sort)
	}
	if params.Order != "" {
		query.Set("order", params.Order)
	}
	perPage := params.PerPage
	if perPage <= 0 {
		perPage = 30
	}
	query.Set("per_page", fmt.Sprintf("%d", perPage))
	if params.Page > 1 {
		query.Set("page", fmt.Sprintf("%d", params.Page))
	}

	path := "/search/issues?" + query.Encode()

	var ghResp struct {
		TotalCount        int  `json:"total_count"`
		IncompleteResults bool `json:"incomplete_results"`
		Items             []struct {
			Number  int    `json:"number"`
			Title   string `json:"title"`
			State   string `json:"state"`
			HTMLURL string `json:"html_url"`
			Body    string `json:"body"`
			User    struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"items"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
