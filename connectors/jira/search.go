package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxSearchResults caps the max_results parameter to prevent
// unbounded result sets from the Jira search API.
const maxSearchResults = 1000

// searchAction implements connectors.Action for jira.search.
// It searches issues using JQL via POST /rest/api/3/search.
type searchAction struct {
	conn *JiraConnector
}

// searchParams holds the validated parameters for a Jira JQL search.
type searchParams struct {
	JQL        string   `json:"jql"`
	MaxResults int      `json:"max_results"`
	Fields     []string `json:"fields"`
}

// validate trims whitespace from the JQL query, checks that it is non-empty,
// and ensures max_results does not exceed the safety cap.
func (p *searchParams) validate() error {
	p.JQL = strings.TrimSpace(p.JQL)
	if p.JQL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: jql"}
	}
	if p.MaxResults > maxSearchResults {
		return &connectors.ValidationError{Message: fmt.Sprintf("max_results cannot exceed %d", maxSearchResults)}
	}
	return nil
}

// Execute runs a JQL search against POST /rest/api/3/search and returns
// the raw Jira response (issues, total, pagination metadata).
func (a *searchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	if params.MaxResults <= 0 {
		params.MaxResults = 50
	}

	body := map[string]interface{}{
		"jql":        params.JQL,
		"maxResults": params.MaxResults,
	}
	if len(params.Fields) > 0 {
		body["fields"] = params.Fields
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/search", body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
