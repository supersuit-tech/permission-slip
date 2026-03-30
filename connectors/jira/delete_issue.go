package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// deleteIssueAction implements connectors.Action for jira.delete_issue.
// It deletes an issue via DELETE /rest/api/3/issue/{issueKey}.
type deleteIssueAction struct {
	conn *JiraConnector
}

type deleteIssueParams struct {
	IssueKey string `json:"issue_key"`
}

func (p *deleteIssueParams) validate() error {
	p.IssueKey = strings.TrimSpace(p.IssueKey)
	if p.IssueKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_key (e.g. PROJ-123)"}
	}
	return nil
}

func (a *deleteIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/issue/" + url.PathEscape(params.IssueKey)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"issue_key": params.IssueKey,
		"status":    "deleted",
	})
}
