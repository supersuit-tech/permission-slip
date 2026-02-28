// Package github implements the GitHub connector for the Permission Slip
// connector execution layer. It uses the GitHub REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultTimeout = 30 * time.Second
)

// GitHubConnector owns the shared HTTP client and base URL used by all
// GitHub actions. Actions hold a pointer back to the connector to access
// these shared resources.
type GitHubConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a GitHubConnector with sensible defaults (30s timeout,
// https://api.github.com base URL).
func New() *GitHubConnector {
	return &GitHubConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a GitHubConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *GitHubConnector {
	return &GitHubConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "github", matching the connectors.id in the database.
func (c *GitHubConnector) ID() string { return "github" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *GitHubConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "github",
		Name:        "GitHub",
		Description: "GitHub integration for repository management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "github.create_issue",
				Name:        "Create Issue",
				Description: "Create a new issue in a repository",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "title"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"title": {
							"type": "string",
							"description": "Issue title"
						},
						"body": {
							"type": "string",
							"description": "Issue body (Markdown supported)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.merge_pr",
				Name:        "Merge Pull Request",
				Description: "Merge an open pull request",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number"
						},
						"merge_method": {
							"type": "string",
							"enum": ["merge", "squash", "rebase"],
							"default": "merge",
							"description": "Merge strategy to use"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "github", AuthType: "api_key", InstructionsURL: "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_github_create_issue_all",
				ActionType:  "github.create_issue",
				Name:        "Create issues (all fields open)",
				Description: "Agent can create issues in any repo with any title and body.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","title":"*","body":"*"}`),
			},
			{
				ID:          "tpl_github_create_issue_org",
				ActionType:  "github.create_issue",
				Name:        "Create issues in your org",
				Description: "Restricts the owner to your organization pattern. Agent can choose the repo, title, and body.",
				Parameters:  json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","title":"*","body":"*"}`),
			},
			{
				ID:          "tpl_github_merge_pr",
				ActionType:  "github.merge_pr",
				Name:        "Merge pull requests",
				Description: "Agent can merge any PR. Owner, repo, and PR number are agent-controlled.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","merge_method":"squash"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *GitHubConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"github.create_issue": &createIssueAction{conn: c},
		"github.merge_pr":     &mergePRAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key, which is required for all GitHub API calls.
func (c *GitHubConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// do is the shared request lifecycle for all GitHub actions. It marshals
// reqBody as JSON, sends the request with auth headers, checks the response
// status, and unmarshals the response into respBody. Either reqBody or
// respBody may be nil (e.g., DELETE with no body, or a request where the
// caller doesn't need the response).
func (c *GitHubConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("GitHub API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("GitHub API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing GitHub response: %v", err)}
		}
	}
	return nil
}

