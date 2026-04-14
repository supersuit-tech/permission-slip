package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// closePRAction implements connectors.Action for github.close_pr.
// It closes a pull request via PATCH /repos/{owner}/{repo}/pulls/{pull_number}.
type closePRAction struct {
	conn *GitHubConnector
}

type closePRParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pull_number"`
}

func (p *closePRParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	return requirePositiveInt(p.PullNumber, "pull_number")
}

// Execute closes an open pull request without merging.
func (a *closePRAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[closePRParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]string{"state": "closed"}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.PullNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, apiPath, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
