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

// assignIssueAction implements connectors.Action for jira.assign_issue.
// It assigns an issue to a user via PUT /rest/api/3/issue/{issueKey}/assignee.
type assignIssueAction struct {
	conn *JiraConnector
}

type assignIssueParams struct {
	IssueKey  string `json:"issue_key"`
	AccountID string `json:"account_id"`
}

func (p *assignIssueParams) validate() error {
	p.IssueKey = strings.TrimSpace(p.IssueKey)
	p.AccountID = strings.TrimSpace(p.AccountID)
	if p.IssueKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_key (e.g. PROJ-123)"}
	}
	if p.AccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: account_id (Atlassian account ID)"}
	}
	return nil
}

func (a *assignIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params assignIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]string{"accountId": params.AccountID}
	path := "/issue/" + url.PathEscape(params.IssueKey) + "/assignee"

	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"issue_key":  params.IssueKey,
		"account_id": params.AccountID,
		"status":     "assigned",
	})
}
