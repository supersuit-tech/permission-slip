package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createBranchAction implements connectors.Action for github.create_branch.
// It creates a branch by first resolving a ref to a SHA, then creating
// a new git ref via POST /repos/{owner}/{repo}/git/refs.
type createBranchAction struct {
	conn *GitHubConnector
}

type createBranchParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BranchName string `json:"branch_name"`
	FromRef    string `json:"from_ref"`
}

func (p *createBranchParams) validate() error {
	if p.Owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if p.Repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	if p.BranchName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: branch_name"}
	}
	if p.FromRef == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from_ref"}
	}
	return nil
}

// Execute creates a new branch in a GitHub repository.
func (a *createBranchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createBranchParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Normalize from_ref: accept bare branch names like "main" and
	// expand to "heads/main" for the Git refs API. Tags can be specified
	// as "tags/v1.0". Full refs like "heads/main" pass through unchanged.
	fromRef := params.FromRef
	if !strings.Contains(fromRef, "/") {
		fromRef = "heads/" + fromRef
	}

	// Escape each segment of the ref path to prevent URL injection.
	// Refs like "heads/main" become "heads/main" (no-op for safe chars),
	// but malicious values like "../foo" are safely encoded per-segment.
	segments := strings.Split(fromRef, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	escapedRef := strings.Join(segments, "/")

	// Resolve the source ref to a SHA.
	var refResp struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}

	refPath := fmt.Sprintf("/repos/%s/%s/git/ref/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), escapedRef)
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, refPath, nil, &refResp); err != nil {
		return nil, err
	}

	// Create the new branch ref.
	ghBody := map[string]string{
		"ref": "refs/heads/" + params.BranchName,
		"sha": refResp.Object.SHA,
	}

	var ghResp struct {
		Ref    string `json:"ref"`
		URL    string `json:"url"`
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}

	path := fmt.Sprintf("/repos/%s/%s/git/refs", url.PathEscape(params.Owner), url.PathEscape(params.Repo))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, ghBody, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
