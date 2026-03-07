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

// addCommentAction implements connectors.Action for jira.add_comment.
// It adds a comment to an issue via POST /rest/api/3/issue/{issueKey}/comment.
type addCommentAction struct {
	conn *JiraConnector
}

type addCommentParams struct {
	IssueKey string `json:"issue_key"`
	Body     string `json:"body"`
}

func (p *addCommentParams) validate() error {
	p.IssueKey = strings.TrimSpace(p.IssueKey)
	if p.IssueKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_key (e.g. PROJ-123)"}
	}
	if strings.TrimSpace(p.Body) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"body": plainTextToADF(params.Body),
	}

	var resp struct {
		ID      string `json:"id"`
		Self    string `json:"self"`
		Created string `json:"created"`
	}

	path := "/issue/" + url.PathEscape(params.IssueKey) + "/comment"
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}

// plainTextToADF wraps plain text in minimal Atlassian Document Format.
// Newlines are converted to separate paragraphs for readability.
func plainTextToADF(text string) map[string]interface{} {
	lines := strings.Split(text, "\n")
	var paragraphs []interface{}

	for _, line := range lines {
		paragraphs = append(paragraphs, map[string]interface{}{
			"type": "paragraph",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": line,
				},
			},
		})
	}

	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": paragraphs,
	}
}
