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
	Org        string `json:"org"`
	Type       string `json:"type"`
	Sort       string `json:"sort"`
	Visibility string `json:"visibility"`
	PerPage    int    `json:"per_page"`
	Page       int    `json:"page"`
}

func (p *listReposParams) validate() error {
	if p.Type != "" {
		switch p.Type {
		case "all", "public", "private", "forks", "sources", "member":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid type: %q; must be one of: all, public, private, forks, sources, member", p.Type)}
		}
	}
	if p.Visibility != "" {
		switch p.Visibility {
		case "all", "public", "private":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid visibility: %q; must be one of: all, public, private", p.Visibility)}
		}
	}
	if p.Sort != "" {
		switch p.Sort {
		case "created", "updated", "pushed", "full_name":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort: %q; must be one of: created, updated, pushed, full_name", p.Sort)}
		}
	}
	if err := validatePerPage(p.PerPage); err != nil {
		return err
	}
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
	if params.Visibility != "" {
		query.Set("visibility", params.Visibility)
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

	var ghResp []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		Private     bool   `json:"private"`
		HTMLURL     string `json:"html_url"`
		Fork        bool   `json:"fork"`
		Description string `json:"description"`
		Language    string `json:"language"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
