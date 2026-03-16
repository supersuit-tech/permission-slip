package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createRepoAction implements connectors.Action for github.create_repo.
// It creates a new repository via POST /user/repos (personal) or
// POST /orgs/{org}/repos (organization).
type createRepoAction struct {
	conn *GitHubConnector
}

type createRepoParams struct {
	Name        string `json:"name"`
	Org         string `json:"org"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	AutoInit    bool   `json:"auto_init"`
}

func (p *createRepoParams) validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if err := validateRepoName(p.Name); err != nil {
		return err
	}
	return nil
}

// Execute creates a GitHub repository and returns the created repo data.
func (a *createRepoAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[createRepoParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	ghBody := map[string]any{
		"name":      params.Name,
		"private":   params.Private,
		"auto_init": params.AutoInit,
	}
	if params.Description != "" {
		ghBody["description"] = params.Description
	}

	org := strings.TrimSpace(params.Org)
	var path string
	if org != "" {
		path = fmt.Sprintf("/orgs/%s/repos", url.PathEscape(org))
	} else {
		path = "/user/repos"
	}

	var ghResp struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		Private     bool   `json:"private"`
		HTMLURL     string `json:"html_url"`
		CloneURL    string `json:"clone_url"`
		Description string `json:"description"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
