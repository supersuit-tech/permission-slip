package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteBranchAction implements connectors.Action for github.delete_branch.
// It deletes a branch ref via DELETE /repos/{owner}/{repo}/git/refs/heads/{branch}.
type deleteBranchAction struct {
	conn *GitHubConnector
}

type deleteBranchParams struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BranchName string `json:"branch_name"`
}

func (p *deleteBranchParams) validate() error {
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return err
	}
	return validateRefName(p.BranchName, "branch_name")
}

// Execute deletes a git branch ref in the repository.
func (a *deleteBranchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[deleteBranchParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	segments := strings.Split("heads/"+params.BranchName, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	refPath := strings.Join(segments, "/")

	apiPath := fmt.Sprintf("/repos/%s/%s/git/refs/%s",
		url.PathEscape(params.Owner), url.PathEscape(params.Repo), refPath)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, apiPath, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"owner":       params.Owner,
		"repo":        params.Repo,
		"branch_name": params.BranchName,
	})
}
