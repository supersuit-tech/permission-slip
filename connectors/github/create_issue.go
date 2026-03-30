package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createIssueAction implements connectors.Action for github.create_issue.
// It creates a new issue in a GitHub repository via POST /repos/{owner}/{repo}/issues.
type createIssueAction struct {
	conn *GitHubConnector
}

// createIssueParams are the parameters parsed from ActionRequest.Parameters.
type createIssueParams struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func (p *createIssueParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

// Execute creates a GitHub issue and returns the created issue data.
func (a *createIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[createIssueParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	// Build request body — only include body field if non-empty.
	ghBody := map[string]string{"title": params.Title}
	if params.Body != "" {
		ghBody["body"] = params.Body
	}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	}

	path := fmt.Sprintf("/repos/%s/%s/issues", url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
