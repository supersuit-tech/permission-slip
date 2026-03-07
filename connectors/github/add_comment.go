package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addCommentAction implements connectors.Action for github.add_comment.
// It adds a comment to an issue or PR via POST /repos/{owner}/{repo}/issues/{issue_number}/comments.
type addCommentAction struct {
	conn *GitHubConnector
}

type addCommentParams struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
	Body        string `json:"body"`
}

func (p *addCommentParams) validate() error {
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.IssueNumber <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: issue_number"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// Execute adds a comment to a GitHub issue or pull request.
func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	ghBody := map[string]string{
		"body": params.Body,
	}

	var ghResp struct {
		ID      int    `json:"id"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		Body    string `json:"body"`
	}

	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
