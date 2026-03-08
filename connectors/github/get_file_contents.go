package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getFileContentsAction implements connectors.Action for github.get_file_contents.
// It reads a file from a repository via GET /repos/{owner}/{repo}/contents/{path}.
type getFileContentsAction struct {
	conn *GitHubConnector
}

type getFileContentsParams struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Path  string `json:"path"`
	Ref   string `json:"ref"`
}

func (p *getFileContentsParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if p.Path == "" {
		return &connectors.ValidationError{Message: "missing required parameter: path"}
	}
	return nil
}

// Execute retrieves file contents from a GitHub repository.
func (a *getFileContentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[getFileContentsParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repos/%s/%s/contents/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.Path)
	if params.Ref != "" {
		path += "?ref=" + url.QueryEscape(params.Ref)
	}

	var ghResp struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		SHA     string `json:"sha"`
		Size    int    `json:"size"`
		Content string `json:"content"`
		HTMLURL string `json:"html_url"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
