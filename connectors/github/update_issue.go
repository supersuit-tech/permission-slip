package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateIssueAction implements connectors.Action for github.update_issue.
// It updates an issue via PATCH /repos/{owner}/{repo}/issues/{issue_number}.
type updateIssueAction struct {
	conn *GitHubConnector
}

type updateIssueParams struct {
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
	IssueNumber int      `json:"issue_number"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	State       string   `json:"state"`
	Assignees   []string `json:"assignees"`
}

func (p *updateIssueParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.IssueNumber, "issue_number"); err != nil {
		return err
	}
	if p.State != "" && p.State != "open" && p.State != "closed" {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid state %q: must be open or closed", p.State)}
	}
	return nil
}

// Execute updates fields on a GitHub issue.
func (a *updateIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[updateIssueParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.Title != "" {
		body["title"] = params.Title
	}
	if params.Body != "" {
		body["body"] = params.Body
	}
	if params.State != "" {
		body["state"] = params.State
	}
	if len(params.Assignees) > 0 {
		body["assignees"] = params.Assignees
	}
	if len(body) == 0 {
		return nil, &connectors.ValidationError{Message: "at least one of title, body, state, or assignees must be provided"}
	}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Title   string `json:"title"`
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/issues/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, apiPath, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
