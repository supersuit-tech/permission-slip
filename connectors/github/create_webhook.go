package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createWebhookAction implements connectors.Action for github.create_webhook.
// It creates a repository webhook via POST /repos/{owner}/{repo}/hooks.
type createWebhookAction struct {
	conn *GitHubConnector
}

type createWebhookParams struct {
	Owner       string   `json:"owner"`
	Repo        string   `json:"repo"`
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	ContentType string   `json:"content_type"`
	Secret      string   `json:"secret"`
	Active      *bool    `json:"active"`
}

func (p *createWebhookParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.URL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: url"}
	}
	if err := requireNonEmptyStrings(p.Events, "events"); err != nil {
		return err
	}
	if p.ContentType != "" {
		switch p.ContentType {
		case "json", "form":
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid content_type: %q; must be one of: json, form", p.ContentType)}
		}
	}
	return nil
}

// Execute creates a webhook for a GitHub repository.
func (a *createWebhookAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[createWebhookParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	config := map[string]string{
		"url":          params.URL,
		"content_type": "json",
	}
	if params.ContentType != "" {
		config["content_type"] = params.ContentType
	}
	if params.Secret != "" {
		config["secret"] = params.Secret
	}

	active := true
	if params.Active != nil {
		active = *params.Active
	}

	body := map[string]any{
		"name":   "web",
		"config": config,
		"events": params.Events,
		"active": active,
	}

	var ghResp struct {
		ID      int      `json:"id"`
		URL     string   `json:"url"`
		HTMLURL string   `json:"html_url"`
		Active  bool     `json:"active"`
		Events  []string `json:"events"`
	}

	path := fmt.Sprintf("/repos/%s/%s/hooks", url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
