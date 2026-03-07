package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// transitionIssueAction implements connectors.Action for jira.transition_issue.
// It moves an issue through workflow states via POST /rest/api/3/issue/{issueKey}/transitions.
type transitionIssueAction struct {
	conn *JiraConnector
}

type transitionIssueParams struct {
	IssueKey       string `json:"issue_key"`
	TransitionID   string `json:"transition_id"`
	TransitionName string `json:"transition_name"`
}

func (p *transitionIssueParams) validate() error {
	if p.IssueKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: issue_key"}
	}
	if p.TransitionID == "" && p.TransitionName == "" {
		return &connectors.ValidationError{Message: "one of transition_id or transition_name is required"}
	}
	return nil
}

func (a *transitionIssueAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params transitionIssueParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	basePath := "/issue/" + url.PathEscape(params.IssueKey) + "/transitions"

	transitionID := params.TransitionID
	if transitionID == "" {
		// Look up by name.
		id, err := a.resolveTransitionName(ctx, req.Credentials, basePath, params.TransitionName)
		if err != nil {
			return nil, err
		}
		transitionID = id
	}

	body := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, basePath, body, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"issue_key":     params.IssueKey,
		"transition_id": transitionID,
		"status":        "transitioned",
	})
}

// resolveTransitionName fetches available transitions and finds the ID
// matching the given name (case-insensitive).
func (a *transitionIssueAction) resolveTransitionName(ctx context.Context, creds connectors.Credentials, path, name string) (string, error) {
	var resp struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"transitions"`
	}

	if err := a.conn.do(ctx, creds, http.MethodGet, path, nil, &resp); err != nil {
		return "", err
	}

	for _, t := range resp.Transitions {
		if equalFold(t.Name, name) {
			return t.ID, nil
		}
	}

	return "", &connectors.ValidationError{
		Message: fmt.Sprintf("transition %q not found for this issue", name),
	}
}

// equalFold is a simple case-insensitive string comparison.
func equalFold(a, b string) bool {
	return len(a) == len(b) && foldLower(a) == foldLower(b)
}

func foldLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
