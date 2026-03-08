package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchCodeAction implements connectors.Action for github.search_code.
// It searches code across repositories via GET /search/code.
type searchCodeAction struct {
	conn *GitHubConnector
}

type searchCodeParams struct {
	Q       string `json:"q"`
	Order   string `json:"order"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

func (p *searchCodeParams) validate() error {
	if p.Q == "" {
		return &connectors.ValidationError{Message: "missing required parameter: q"}
	}
	return nil
}

// Execute searches code across GitHub repositories.
func (a *searchCodeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[searchCodeParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("q", params.Q)
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

	path := "/search/code?" + query.Encode()

	var ghResp struct {
		TotalCount        int  `json:"total_count"`
		IncompleteResults bool `json:"incomplete_results"`
		Items             []struct {
			Name    string `json:"name"`
			Path    string `json:"path"`
			HTMLURL string `json:"html_url"`
			SHA     string `json:"sha"`
			Repository struct {
				FullName string `json:"full_name"`
				HTMLURL  string `json:"html_url"`
			} `json:"repository"`
		} `json:"items"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
