package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPRAction implements connectors.Action for github.create_pr.
// It creates a pull request via POST /repos/{owner}/{repo}/pulls.
type createPRAction struct {
	conn *GitHubConnector
}

type createPRParams struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft"`
}

func (p *createPRParams) validate() error {
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if p.Head == "" {
		return &connectors.ValidationError{Message: "missing required parameter: head"}
	}
	if p.Base == "" {
		return &connectors.ValidationError{Message: "missing required parameter: base"}
	}
	return nil
}

// Execute creates a GitHub pull request and returns the created PR data.
func (a *createPRAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPRParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	ghBody := map[string]any{
		"title": params.Title,
		"head":  params.Head,
		"base":  params.Base,
	}
	if params.Body != "" {
		ghBody["body"] = params.Body
	}
	if params.Draft {
		ghBody["draft"] = true
	}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls", url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
