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

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
