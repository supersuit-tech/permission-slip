package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addLabelAction implements connectors.Action for github.add_label.
// It adds labels to an issue or PR via POST /repos/{owner}/{repo}/issues/{issue_number}/labels.
type addLabelAction struct {
	conn *GitHubConnector
}

type addLabelParams struct {
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
	IssueNumber int      `json:"issue_number"`
	Labels      []string `json:"labels"`
}

func (p *addLabelParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.IssueNumber, "issue_number"); err != nil {
		return err
	}
	return requireNonEmptyStrings(p.Labels, "labels")
}

// Execute adds labels to a GitHub issue or pull request.
func (a *addLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[addLabelParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	ghBody := map[string]any{
		"labels": params.Labels,
	}

	var ghResp []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.IssueNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
