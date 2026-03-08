package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// triggerWorkflowAction implements connectors.Action for github.trigger_workflow.
// It dispatches a GitHub Actions workflow via
// POST /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches.
type triggerWorkflowAction struct {
	conn *GitHubConnector
}

type triggerWorkflowParams struct {
	Owner      string         `json:"owner"`
	Repo       string         `json:"repo"`
	WorkflowID string         `json:"workflow_id"`
	Ref        string         `json:"ref"`
	Inputs     map[string]any `json:"inputs"`
}

func (p *triggerWorkflowParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.WorkflowID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: workflow_id"}
	}
	if p.Ref == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ref"}
	}
	return nil
}

// Execute triggers a GitHub Actions workflow dispatch event.
func (a *triggerWorkflowAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[triggerWorkflowParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"ref": params.Ref,
	}
	if len(params.Inputs) > 0 {
		body["inputs"] = params.Inputs
	}

	// GitHub returns 204 No Content on success — pass nil for respBody.
	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/dispatches",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), url.PathEscape(params.WorkflowID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{"status": "dispatched"})
}
