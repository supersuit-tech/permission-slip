package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

// Execute creates a GitHub issue and returns the created issue data.
func (a *createIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
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
