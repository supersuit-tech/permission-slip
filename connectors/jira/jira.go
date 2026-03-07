// Package jira implements the Jira connector for the Permission Slip
// connector execution layer. It uses the Jira Cloud REST API v3 with
// basic auth (email + API token) via plain net/http.
package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validSite matches Atlassian site subdomains: alphanumeric with hyphens.
// Prevents SSRF by ensuring the site value cannot contain path separators,
// fragments, or other characters that would alter the target host.
var validSite = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

const (
	defaultTimeout = 30 * time.Second
	// maxResponseBody is the maximum response body size (10 MB) to prevent OOM
	// from malicious or buggy API responses.
	maxResponseBody = 10 << 20
)

// JiraConnector owns the shared HTTP client used by all Jira actions.
// The base URL is constructed per-request from the site credential
// (https://{site}.atlassian.net/rest/api/3/).
type JiraConnector struct {
	client  *http.Client
	baseURL string // empty for production (derived from credentials); set for tests
}

// New creates a JiraConnector with sensible defaults (30s timeout).
func New() *JiraConnector {
	return &JiraConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a JiraConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *JiraConnector {
	return &JiraConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "jira", matching the connectors.id in the database.
func (c *JiraConnector) ID() string { return "jira" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *JiraConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "jira",
		Name:        "Jira",
		Description: "Jira Cloud integration for issue tracking and project management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "jira.create_issue",
				Name:        "Create Issue",
				Description: "Create a new issue in a Jira project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_key", "issue_type", "summary"],
					"properties": {
						"project_key": {
							"type": "string",
							"description": "Project key (e.g. PROJ)"
						},
						"issue_type": {
							"type": "string",
							"description": "Issue type (e.g. Bug, Story, Task)"
						},
						"summary": {
							"type": "string",
							"description": "Issue summary/title"
						},
						"description": {
							"type": "string",
							"description": "Issue description (plain text, converted to ADF)"
						},
						"assignee": {
							"type": "string",
							"description": "Atlassian account ID of the assignee"
						},
						"priority": {
							"type": "string",
							"description": "Priority name (e.g. High, Medium, Low)"
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to apply to the issue"
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.update_issue",
				Name:        "Update Issue",
				Description: "Update fields on an existing Jira issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)"
						},
						"summary": {
							"type": "string",
							"description": "Updated summary/title"
						},
						"description": {
							"type": "string",
							"description": "Updated description (plain text, converted to ADF)"
						},
						"assignee": {
							"type": "string",
							"description": "Atlassian account ID of the assignee"
						},
						"priority": {
							"type": "string",
							"description": "Priority name (e.g. High, Medium, Low)"
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to set on the issue"
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.transition_issue",
				Name:        "Transition Issue",
				Description: "Move an issue through workflow states",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)"
						},
						"transition_id": {
							"type": "string",
							"description": "Transition ID to apply"
						},
						"transition_name": {
							"type": "string",
							"description": "Transition name (e.g. In Progress, Done). Looked up if transition_id is not provided."
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment to a Jira issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key", "body"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)"
						},
						"body": {
							"type": "string",
							"description": "Comment text (plain text, converted to ADF)"
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.assign_issue",
				Name:        "Assign Issue",
				Description: "Assign an issue to a user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key", "account_id"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)"
						},
						"account_id": {
							"type": "string",
							"description": "Atlassian account ID of the user"
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.search",
				Name:        "Search Issues",
				Description: "Search issues using JQL queries",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["jql"],
					"properties": {
						"jql": {
							"type": "string",
							"description": "JQL query string"
						},
						"max_results": {
							"type": "integer",
							"default": 50,
							"description": "Maximum number of results to return"
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Fields to include in the response"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "jira",
				AuthType:        "basic",
				InstructionsURL: "https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_jira_create_issue_project",
				ActionType:  "jira.create_issue",
				Name:        "Create issues in a project",
				Description: "Agent can create issues in a specific project with any type, summary, and details.",
				Parameters:  json.RawMessage(`{"project_key":"YOUR_PROJECT","issue_type":"*","summary":"*","description":"*","assignee":"*","priority":"*","labels":"*"}`),
			},
			{
				ID:          "tpl_jira_create_issue_all",
				ActionType:  "jira.create_issue",
				Name:        "Create issues (all projects)",
				Description: "Agent can create issues in any project with all fields open.",
				Parameters:  json.RawMessage(`{"project_key":"*","issue_type":"*","summary":"*","description":"*","assignee":"*","priority":"*","labels":"*"}`),
			},
			{
				ID:          "tpl_jira_transition_issue",
				ActionType:  "jira.transition_issue",
				Name:        "Transition issues",
				Description: "Agent can move any issue through workflow states.",
				Parameters:  json.RawMessage(`{"issue_key":"*","transition_id":"*","transition_name":"*"}`),
			},
			{
				ID:          "tpl_jira_search_assigned",
				ActionType:  "jira.search",
				Name:        "Search issues assigned to me",
				Description: "Search for issues assigned to the current user.",
				Parameters:  json.RawMessage(`{"jql":"assignee = currentUser()","max_results":"*","fields":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *JiraConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"jira.create_issue":     &createIssueAction{conn: c},
		"jira.update_issue":     &updateIssueAction{conn: c},
		"jira.transition_issue": &transitionIssueAction{conn: c},
		"jira.add_comment":      &addCommentAction{conn: c},
		"jira.assign_issue":     &assignIssueAction{conn: c},
		"jira.search":           &searchAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain
// non-empty site, email, and api_token fields, which are required for
// all Jira API calls.
func (c *JiraConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return &connectors.ValidationError{Message: "missing required credential: site"}
	}
	email, ok := creds.Get("email")
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "missing required credential: email"}
	}
	token, ok := creds.Get("api_token")
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	return nil
}

// apiBase returns the base URL for Jira REST API v3 calls. In test mode
// it returns the test server URL; in production it builds the URL from
// the site credential.
func (c *JiraConnector) apiBase(creds connectors.Credentials) (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return "", &connectors.ValidationError{Message: "missing required credential: site"}
	}
	if !validSite.MatchString(site) {
		return "", &connectors.ValidationError{
			Message: "invalid site credential: must contain only alphanumeric characters and hyphens (e.g. \"my-company\")",
		}
	}
	return "https://" + site + ".atlassian.net/rest/api/3", nil
}

// do is the shared request lifecycle for all Jira actions. It marshals
// reqBody as JSON, sends the request with basic auth headers, checks the
// response status, and unmarshals the response into respBody. Either
// reqBody or respBody may be nil.
func (c *JiraConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	base, err := c.apiBase(creds)
	if err != nil {
		return err
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	email, ok := creds.Get("email")
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "email credential is missing or empty"}
	}
	token, ok := creds.Get("api_token")
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_token credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(email+":"+token)))

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Jira API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Jira API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Jira response: %v", err)}
		}
	}
	return nil
}
