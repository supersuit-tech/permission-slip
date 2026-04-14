package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// removeLabelAction implements connectors.Action for github.remove_label.
// It removes a label from an issue or PR via
// DELETE /repos/{owner}/{repo}/issues/{issue_number}/labels/{name}.
type removeLabelAction struct {
	conn *GitHubConnector
}

type removeLabelParams struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
	Name        string `json:"name"`
}

func (p *removeLabelParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.IssueNumber, "issue_number"); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

// Execute removes one label from an issue or pull request.
func (a *removeLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[removeLabelParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber,
		url.PathEscape(params.Name))
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, apiPath, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"owner":        params.Owner,
		"repo":         params.Repo,
		"issue_number": params.IssueNumber,
		"name":         params.Name,
	})
}
