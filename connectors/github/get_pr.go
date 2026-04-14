package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getPRAction implements connectors.Action for github.get_pr.
// It fetches a pull request via GET /repos/{owner}/{repo}/pulls/{pull_number}.
type getPRAction struct {
	conn *GitHubConnector
}

type getPRParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pull_number"`
}

func (p *getPRParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	return requirePositiveInt(p.PullNumber, "pull_number")
}

// Execute returns pull request JSON from the GitHub API.
func (a *getPRAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[getPRParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.PullNumber)

	var ghResp map[string]any
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiPath, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
