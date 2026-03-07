package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.IssueNumber <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: issue_number"}
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
	var params closeIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
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
