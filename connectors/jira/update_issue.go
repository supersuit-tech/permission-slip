package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateIssueAction implements connectors.Action for jira.update_issue.
// It updates fields on an existing issue via PUT /rest/api/3/issue/{issueKey}.
type updateIssueAction struct {
	conn *JiraConnector
}

type updateIssueParams struct {
	IssueKey    string   `json:"issue_key"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Assignee    string   `json:"assignee"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
}

func (p *updateIssueParams) validate() error {
	p.IssueKey = strings.TrimSpace(p.IssueKey)
	if p.IssueKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_key (e.g. PROJ-123)"}
	}
	return nil
}

func (a *updateIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	fields := map[string]interface{}{}
	if params.Summary != "" {
		fields["summary"] = params.Summary
	}
	if params.Description != "" {
		fields["description"] = plainTextToADF(params.Description)
	}
	if params.Assignee != "" {
		fields["assignee"] = map[string]string{"accountId": params.Assignee}
	}
	if params.Priority != "" {
		fields["priority"] = map[string]string{"name": params.Priority}
	}
	if params.Labels != nil {
		fields["labels"] = params.Labels
	}

	body := map[string]interface{}{"fields": fields}
	path := "/issue/" + url.PathEscape(params.IssueKey)

	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, nil); err != nil {
		return nil, err
	}

	// Jira returns 204 No Content on successful update.
	return connectors.JSONResult(map[string]string{"issue_key": params.IssueKey, "status": "updated"})
}
