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
	if p.Sort != "" {
		switch p.Sort {
		case "comments", "reactions", "reactions-+1", "reactions--1", "reactions-smile",
			"reactions-thinking_face", "reactions-heart", "reactions-tada",
			"interactions", "created", "updated":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort: %q; must be one of: comments, reactions, reactions-+1, reactions--1, reactions-smile, reactions-thinking_face, reactions-heart, reactions-tada, interactions, created, updated", p.Sort)}
		}
	}
	if p.Order != "" {
		switch p.Order {
		case "asc", "desc":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid order: %q; must be one of: asc, desc", p.Order)}
		}
	}
	if err := validatePerPage(p.PerPage); err != nil {
		return err
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
	setPagination(query, params.PerPage, params.Page)

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
