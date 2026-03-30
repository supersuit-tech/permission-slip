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

	"github.com/supersuit-tech/permission-slip/connectors"
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

// Actions returns the registered action handlers keyed by action_type.
func (c *GitHubConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"github.create_issue":          &createIssueAction{conn: c},
		"github.merge_pr":              &mergePRAction{conn: c},
		"github.create_pr":             &createPRAction{conn: c},
		"github.add_reviewer":          &addReviewerAction{conn: c},
		"github.create_release":        &createReleaseAction{conn: c},
		"github.close_issue":           &closeIssueAction{conn: c},
		"github.add_label":             &addLabelAction{conn: c},
		"github.add_comment":           &addCommentAction{conn: c},
		"github.create_branch":         &createBranchAction{conn: c},
		"github.get_file_contents":     &getFileContentsAction{conn: c},
		"github.create_or_update_file": &createOrUpdateFileAction{conn: c},
		"github.list_repos":            &listReposAction{conn: c},
		"github.get_repo":              &getRepoAction{conn: c},
		"github.list_pull_requests":    &listPullRequestsAction{conn: c},
		"github.trigger_workflow":      &triggerWorkflowAction{conn: c},
		"github.list_workflow_runs":    &listWorkflowRunsAction{conn: c},
		"github.create_webhook":        &createWebhookAction{conn: c},
		"github.search_code":           &searchCodeAction{conn: c},
		"github.search_issues":         &searchIssuesAction{conn: c},
		"github.create_repo":           &createRepoAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either a
// non-empty access_token (from OAuth) or a non-empty api_key (PAT). OAuth
// tokens take precedence when both are present.
func (c *GitHubConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get("access_token"); ok && token != "" {
		return nil
	}
	if key, ok := creds.Get("api_key"); ok && key != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token or api_key"}
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

	token := ""
	if t, ok := creds.Get("access_token"); ok && t != "" {
		token = t
	} else if k, ok := creds.Get("api_key"); ok && k != "" {
		token = k
	}
	if token == "" {
		return &connectors.ValidationError{Message: "access_token or api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+token)

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
