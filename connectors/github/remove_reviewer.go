package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// removeReviewerAction implements connectors.Action for github.remove_reviewer.
// It removes requested reviewers via
// DELETE /repos/{owner}/{repo}/pulls/{pull_number}/requested_reviewers.
type removeReviewerAction struct {
	conn *GitHubConnector
}

type removeReviewerParams struct {
	Owner      string   `json:"owner"`
	Repo       string   `json:"repo"`
	PullNumber int      `json:"pull_number"`
	Reviewers  []string `json:"reviewers"`
}

func (p *removeReviewerParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.PullNumber, "pull_number"); err != nil {
		return err
	}
	return requireNonEmptyStrings(p.Reviewers, "reviewers")
}

// Execute removes requested reviewers from a pull request.
func (a *removeReviewerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[removeReviewerParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]any{"reviewers": params.Reviewers}

	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d/requested_reviewers",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.PullNumber)

	var ghResp map[string]any
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, apiPath, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
