package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.IssueNumber, "issue_number"); err != nil {
		return err
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// Execute adds a comment to a GitHub issue or pull request.
func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[addCommentParams](req.Parameters)
	if err != nil {
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
