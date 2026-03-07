package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createIssueAction implements connectors.Action for jira.create_issue.
// It creates a new issue via POST /rest/api/3/issue.
type createIssueAction struct {
	conn *JiraConnector
}

type createIssueParams struct {
	ProjectKey  string   `json:"project_key"`
	IssueType   string   `json:"issue_type"`
	Summary     string   `json:"summary"`
	Description string   `json:"description"`
	Assignee    string   `json:"assignee"`
	Priority    string   `json:"priority"`
	Labels      []string `json:"labels"`
}

func (p *createIssueParams) validate() error {
	if p.ProjectKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_key"}
	}
	if p.IssueType == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_type"}
	}
	if p.Summary == "" {
		return &connectors.ValidationError{Message: "missing required parameter: summary"}
	}
	return nil
}

func (a *createIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"project":   map[string]string{"key": params.ProjectKey},
		"issuetype": map[string]string{"name": params.IssueType},
		"summary":   params.Summary,
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
	if len(params.Labels) > 0 {
		fields["labels"] = params.Labels
	}

	body := map[string]interface{}{"fields": fields}

	var resp struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/issue", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
