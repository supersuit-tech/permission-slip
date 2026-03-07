package github

import (
	"context"
	"encoding/json"
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
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.IssueNumber <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: issue_number"}
	}
	if len(p.Labels) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: labels"}
	}
	return nil
}

// Execute adds labels to a GitHub issue or pull request.
func (a *addLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addLabelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
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
