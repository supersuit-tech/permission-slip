package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteFileAction implements connectors.Action for github.delete_file.
// It deletes a file via DELETE /repos/{owner}/{repo}/contents/{path}.
type deleteFileAction struct {
	conn *GitHubConnector
}

type deleteFileParams struct {
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	Message string `json:"message"`
	SHA     string `json:"sha"`
	Branch  string `json:"branch"`
}

func (p *deleteFileParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.Path == "" {
		return &connectors.ValidationError{Message: "missing required parameter: path"}
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	if p.SHA == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sha (blob SHA of the file to delete)"}
	}
	if err := validateFilePath(p.Path); err != nil {
		return err
	}
	return nil
}

// Execute deletes a file from a GitHub repository.
func (a *deleteFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[deleteFileParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]string{
		"message": params.Message,
		"sha":     params.SHA,
	}
	if params.Branch != "" {
		body["branch"] = params.Branch
	}

	var ghResp struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}

	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), escapeFilePath(params.Path))
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, apiPath, body, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
