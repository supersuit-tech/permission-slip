package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

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
	if err := validateFilePath(p.Path); err != nil {
		return err
	}
	return nil
}

// Execute retrieves file contents from a GitHub repository.
// The response includes both the raw base64 content from GitHub and a
// decoded_content field with the UTF-8 text (empty for binary files).
func (a *getFileContentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[getFileContentsParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repos/%s/%s/contents/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), escapeFilePath(params.Path))
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

	// Decode the base64 content for AI agents that need to read the file as text.
	// GitHub wraps base64 at 60 chars — strip whitespace before decoding.
	decodedContent := ""
	if ghResp.Content != "" {
		b64 := strings.ReplaceAll(ghResp.Content, "\n", "")
		if decoded, err := base64.StdEncoding.DecodeString(b64); err == nil && utf8.Valid(decoded) {
			decodedContent = string(decoded)
		}
	}

	result := struct {
		Name           string `json:"name"`
		Path           string `json:"path"`
		SHA            string `json:"sha"`
		Size           int    `json:"size"`
		Content        string `json:"content"`
		DecodedContent string `json:"decoded_content"`
		HTMLURL        string `json:"html_url"`
	}{
		Name:           ghResp.Name,
		Path:           ghResp.Path,
		SHA:            ghResp.SHA,
		Size:           ghResp.Size,
		Content:        ghResp.Content,
		DecodedContent: decodedContent,
		HTMLURL:        ghResp.HTMLURL,
	}

	return connectors.JSONResult(result)
}
