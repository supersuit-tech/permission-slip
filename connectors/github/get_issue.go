package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getIssueAction implements connectors.Action for github.get_issue.
// It fetches an issue via GET /repos/{owner}/{repo}/issues/{issue_number}.
type getIssueAction struct {
	conn *GitHubConnector
}

type getIssueParams struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
}

func (p *getIssueParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	return requirePositiveInt(p.IssueNumber, "issue_number")
}

// Execute returns issue JSON from the GitHub API.
func (a *getIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[getIssueParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/issues/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)

	var ghResp map[string]any
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, apiPath, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
