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
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.State != "" {
		switch p.State {
		case "open", "closed", "all":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid state: %q; must be one of: open, closed, all", p.State)}
		}
	}
	if p.Sort != "" {
		switch p.Sort {
		case "created", "updated", "popularity", "long-running":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort: %q; must be one of: created, updated, popularity, long-running", p.Sort)}
		}
	}
	if p.Direction != "" {
		switch p.Direction {
		case "asc", "desc":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid direction: %q; must be one of: asc, desc", p.Direction)}
		}
	}
	if err := validatePerPage(p.PerPage); err != nil {
		return err
	}
	return nil
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
	setPagination(query, params.PerPage, params.Page)

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
