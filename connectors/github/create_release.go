package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createReleaseAction implements connectors.Action for github.create_release.
// It creates a release via POST /repos/{owner}/{repo}/releases.
type createReleaseAction struct {
	conn *GitHubConnector
}

type createReleaseParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func (p *createReleaseParams) validate() error {
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.TagName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tag_name"}
	}
	return nil
}

// Execute creates a GitHub release and returns the created release data.
func (a *createReleaseAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createReleaseParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	ghBody := map[string]any{
		"tag_name": params.TagName,
	}
	if params.Name != "" {
		ghBody["name"] = params.Name
	}
	if params.Body != "" {
		ghBody["body"] = params.Body
	}
	if params.Draft {
		ghBody["draft"] = true
	}
	if params.Prerelease {
		ghBody["prerelease"] = true
	}

	var ghResp struct {
		ID      int    `json:"id"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Draft   bool   `json:"draft"`
	}

	path := fmt.Sprintf("/repos/%s/%s/releases", url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
