package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updatePRAction implements connectors.Action for github.update_pr.
// It updates a pull request via PATCH /repos/{owner}/{repo}/pulls/{pull_number}.
type updatePRAction struct {
	conn *GitHubConnector
}

type updatePRParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	PullNumber int    `json:"pull_number"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	Base       string `json:"base"`
	State      string `json:"state"`
}

func (p *updatePRParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.PullNumber, "pull_number"); err != nil {
		return err
	}
	if p.State != "" && p.State != "open" && p.State != "closed" {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid state %q: must be open or closed", p.State)}
	}
	return nil
}

// Execute updates fields on a GitHub pull request.
func (a *updatePRAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[updatePRParams](req.Parameters)
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
	if params.Base != "" {
		body["base"] = params.Base
	}
	if params.State != "" {
		body["state"] = params.State
	}
	if len(body) == 0 {
		return nil, &connectors.ValidationError{Message: "at least one of title, body, base, or state must be provided"}
	}

	var ghResp struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		Title   string `json:"title"`
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/pulls/%d",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.PullNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, apiPath, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
