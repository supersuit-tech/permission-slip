package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// addReviewerAction implements connectors.Action for github.add_reviewer.
// It requests reviews on a PR via POST /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers.
type addReviewerAction struct {
	conn *GitHubConnector
}

type addReviewerParams struct {
	Owner      string   `json:"owner"`
	Repo       string   `json:"repo"`
	PullNumber int      `json:"pull_number"`
	Reviewers  []string `json:"reviewers"`
}

func (p *addReviewerParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.PullNumber, "pull_number"); err != nil {
		return err
	}
	return requireNonEmptyStrings(p.Reviewers, "reviewers")
}

// Execute requests reviews on a pull request and returns the updated PR data.
func (a *addReviewerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[addReviewerParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	ghBody := map[string]any{
		"reviewers": params.Reviewers,
	}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/requested_reviewers",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.PullNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
