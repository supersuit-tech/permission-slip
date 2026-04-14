package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listIssuesAction implements connectors.Action for github.list_issues.
// It lists issues via GET /repos/{owner}/{repo}/issues.
type listIssuesAction struct {
	conn *GitHubConnector
}

type listIssuesParams struct {
	Owner               string `json:"owner"`
	Repo                string `json:"repo"`
	State               string `json:"state"`
	Labels              string `json:"labels"`
	Sort                string `json:"sort"`
	Direction           string `json:"direction"`
	Since               string `json:"since"`
	IncludePullRequests bool   `json:"include_pull_requests"`
	PerPage             int    `json:"per_page"`
	Page                int    `json:"page"`
}

func (p *listIssuesParams) validate() error {
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
		case "created", "updated", "comments":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort: %q; must be one of: created, updated, comments", p.Sort)}
		}
	}
	if p.Direction != "" {
		switch p.Direction {
		case "asc", "desc":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid direction: %q; must be one of: asc, desc", p.Direction)}
		}
	}
	return validatePerPage(p.PerPage)
}

// Execute lists issues for a repository (excludes pull requests by default per GitHub API).
func (a *listIssuesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listIssuesParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.State != "" {
		q.Set("state", params.State)
	}
	if params.Labels != "" {
		q.Set("labels", params.Labels)
	}
	if params.Sort != "" {
		q.Set("sort", params.Sort)
	}
	if params.Direction != "" {
		q.Set("direction", params.Direction)
	}
	if params.Since != "" {
		q.Set("since", params.Since)
	}
	setPagination(q, params.PerPage, params.Page)

	apiPath := fmt.Sprintf("/repos/%s/%s/issues?%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), q.Encode())

	var ghResp []map[string]any
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiPath, nil, &ghResp); err != nil {
		return nil, err
	}

	if !params.IncludePullRequests {
		filtered := ghResp[:0]
		for _, item := range ghResp {
			if _, isPR := item["pull_request"]; isPR {
				continue
			}
			filtered = append(filtered, item)
		}
		ghResp = filtered
	}

	return connectors.JSONResult(ghResp)
}
