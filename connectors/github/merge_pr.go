package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// mergePRAction implements connectors.Action for github.merge_pr.
// It merges a pull request via PUT /repos/{owner}/{repo}/pulls/{pull_number}/merge.
type mergePRAction struct {
	conn *GitHubConnector
}

// mergePRParams are the parameters parsed from ActionRequest.Parameters.
type mergePRParams struct {
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	PullNumber  int    `json:"pull_number"`
	MergeMethod string `json:"merge_method"`
}

var validMergeMethods = map[string]bool{
	"merge":  true,
	"squash": true,
	"rebase": true,
}

func (p *mergePRParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	if err := requirePositiveInt(p.PullNumber, "pull_number"); err != nil {
		return err
	}
	if p.MergeMethod == "" {
		p.MergeMethod = "merge"
	}
	if !validMergeMethods[p.MergeMethod] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid merge_method %q: must be merge, squash, or rebase", p.MergeMethod)}
	}
	return nil
}

// Execute merges a GitHub pull request and returns the merge result.
func (a *mergePRAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[mergePRParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	var ghResp struct {
		SHA     string `json:"sha"`
		Merged  bool   `json:"merged"`
		Message string `json:"message"`
	}

	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", url.PathEscape(params.Owner), url.PathEscape(params.Repo), params.PullNumber)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, map[string]string{"merge_method": params.MergeMethod}, &ghResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(ghResp)
}
