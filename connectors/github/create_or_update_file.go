package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createOrUpdateFileAction implements connectors.Action for github.create_or_update_file.
// It creates or updates a file via PUT /repos/{owner}/{repo}/contents/{path}.
type createOrUpdateFileAction struct {
	conn *GitHubConnector
}

type createOrUpdateFileParams struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha"`
}

func (p *createOrUpdateFileParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.Path == "" {
		return &connectors.ValidationError{Message: "missing required parameter: path"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	return nil
}

// Execute creates or updates a file in a GitHub repository.
func (a *createOrUpdateFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[createOrUpdateFileParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]string{
		"message": params.Message,
		"content": params.Content,
	}
	if params.Branch != "" {
		body["branch"] = params.Branch
	}
	if params.SHA != "" {
		body["sha"] = params.SHA
	}

	var ghResp struct {
		Content struct {
			Name    string `json:"name"`
			Path    string `json:"path"`
			SHA     string `json:"sha"`
			HTMLURL string `json:"html_url"`
		} `json:"content"`
		Commit struct {
			SHA     string `json:"sha"`
			Message string `json:"message"`
		} `json:"commit"`
	}

	path := fmt.Sprintf("/repos/%s/%s/contents/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.Path)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
