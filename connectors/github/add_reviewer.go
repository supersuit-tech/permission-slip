package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.PullNumber <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: pull_number"}
	}
	if len(p.Reviewers) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: reviewers"}
	}
	for _, r := range p.Reviewers {
		if r == "" {
			return &connectors.ValidationError{Message: "reviewers must not contain empty strings"}
		}
	}
	return nil
}

// Execute requests reviews on a pull request and returns the updated PR data.
func (a *addReviewerAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addReviewerParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
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
