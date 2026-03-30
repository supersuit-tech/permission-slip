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

// getIssueAction implements connectors.Action for jira.get_issue.
// It retrieves a single issue via GET /rest/api/3/issue/{issueKey}.
type getIssueAction struct {
	conn *JiraConnector
}

type getIssueParams struct {
	IssueKey string   `json:"issue_key"`
	Fields   []string `json:"fields"`
}

func (p *getIssueParams) validate() error {
	p.IssueKey = strings.TrimSpace(p.IssueKey)
	if p.IssueKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_key (e.g. PROJ-123)"}
	}
	return nil
}

func (a *getIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/issue/" + url.PathEscape(params.IssueKey)
	if len(params.Fields) > 0 {
		path += "?fields=" + url.QueryEscape(strings.Join(params.Fields, ","))
	}

	var resp jiraIssueResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"id":      resp.ID,
		"key":     resp.Key,
		"self":    resp.Self,
		"summary": resp.Fields.Summary,
		"created": resp.Fields.Created,
		"updated": resp.Fields.Updated,
	}
	if resp.Fields.Status != nil {
		result["status"] = resp.Fields.Status.Name
	}
	if resp.Fields.Assignee != nil {
		result["assignee"] = resp.Fields.Assignee.DisplayName
		result["assignee_account_id"] = resp.Fields.Assignee.AccountID
	}
	if resp.Fields.Priority != nil {
		result["priority"] = resp.Fields.Priority.Name
	}
	if resp.Fields.IssueType != nil {
		result["issue_type"] = resp.Fields.IssueType.Name
	}
	if len(resp.Fields.Labels) > 0 {
		result["labels"] = resp.Fields.Labels
	}

	return connectors.JSONResult(result)
}

type jiraIssueResponse struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields struct {
		Summary string   `json:"summary"`
		Created string   `json:"created"`
		Updated string   `json:"updated"`
		Labels  []string `json:"labels"`
		Status  *struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee *struct {
			DisplayName string `json:"displayName"`
			AccountID   string `json:"accountId"`
		} `json:"assignee"`
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
		IssueType *struct {
			Name string `json:"name"`
		} `json:"issuetype"`
	} `json:"fields"`
}
