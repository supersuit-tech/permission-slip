package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteWebhookAction implements connectors.Action for github.delete_webhook.
// It deletes a repository webhook via DELETE /repos/{owner}/{repo}/hooks/{hook_id}.
type deleteWebhookAction struct {
	conn *GitHubConnector
}

type deleteWebhookParams struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	HookID int    `json:"hook_id"`
}

func (p *deleteWebhookParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	return requirePositiveInt(p.HookID, "hook_id")
}

// Execute deletes a repository webhook.
func (a *deleteWebhookAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[deleteWebhookParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/hooks/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.HookID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, apiPath, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"owner":   params.Owner,
		"repo":    params.Repo,
		"hook_id": params.HookID,
	})
}
