package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// closeIssueAction implements connectors.Action for github.close_issue.
// It closes an issue via PATCH /repos/{owner}/{repo}/issues/{issue_number},
// optionally adding a comment first.
type closeIssueAction struct {
	conn *GitHubConnector
}

type closeIssueParams struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
	StateReason string `json:"state_reason"`
	Comment     string `json:"comment"`
}

var validStateReasons = map[string]bool{
	"completed":   true,
	"not_planned": true,
}

func (p *closeIssueParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.IssueNumber, "issue_number"); err != nil {
		return err
	}
	if p.StateReason == "" {
		p.StateReason = "completed"
	}
	if !validStateReasons[p.StateReason] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid state_reason %q: must be completed or not_planned", p.StateReason)}
	}
	return nil
}

// Execute closes a GitHub issue, optionally posting a comment first.
func (a *closeIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[closeIssueParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	// If a comment is provided, post it before closing.
	if params.Comment != "" {
		commentPath := fmt.Sprintf("/repos/%s/%s/issues/%d/comments",
			url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)
		if err := a.conn.do(ctx, req.Credentials, http.MethodPost, commentPath,
			map[string]string{"body": params.Comment}, nil); err != nil {
			return nil, err
		}
	}

	// Close the issue.
	ghBody := map[string]string{
		"state":        "closed",
		"state_reason": params.StateReason,
	}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
	}

	path := fmt.Sprintf("/repos/%s/%s/issues/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
