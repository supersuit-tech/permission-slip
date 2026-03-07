// Package linear implements the Linear connector for the Permission Slip
// connector execution layer. It uses the Linear GraphQL API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.linear.app/graphql"
	defaultTimeout = 30 * time.Second
	credKeyAPIKey  = "api_key"

	// defaultRetryAfter is used when Linear returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes limits how much data we read from the Linear API
	// to prevent OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// LinearConnector owns the shared HTTP client and base URL used by all
// Linear actions. Actions hold a pointer back to the connector to access
// these shared resources.
type LinearConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a LinearConnector with sensible defaults (30s timeout,
// https://api.linear.app/graphql base URL).
func New() *LinearConnector {
	return &LinearConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a LinearConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *LinearConnector {
	return &LinearConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "linear", matching the connectors.id in the database.
func (c *LinearConnector) ID() string { return "linear" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *LinearConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "linear",
		Name:        "Linear",
		Description: "Linear integration for issue tracking and project management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "linear.create_issue",
				Name:        "Create Issue",
				Description: "Create a new issue in a Linear team",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id", "title"],
					"properties": {
						"team_id": {
							"type": "string",
							"description": "The team ID to create the issue in"
						},
						"title": {
							"type": "string",
							"description": "Issue title"
						},
						"description": {
							"type": "string",
							"description": "Issue description (markdown)"
						},
						"assignee_id": {
							"type": "string",
							"description": "User ID to assign the issue to"
						},
						"priority": {
							"type": "integer",
							"minimum": 0,
							"maximum": 4,
							"description": "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low"
						},
						"state_id": {
							"type": "string",
							"description": "Workflow state ID"
						},
						"label_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Label IDs to apply"
						},
						"project_id": {
							"type": "string",
							"description": "Project ID to associate with"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.update_issue",
				Name:        "Update Issue",
				Description: "Update fields on an existing Linear issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_id"],
					"properties": {
						"issue_id": {
							"type": "string",
							"description": "The issue ID to update"
						},
						"title": {
							"type": "string",
							"description": "New issue title"
						},
						"description": {
							"type": "string",
							"description": "New issue description (markdown)"
						},
						"assignee_id": {
							"type": "string",
							"description": "User ID to assign the issue to"
						},
						"priority": {
							"type": "integer",
							"minimum": 0,
							"maximum": 4,
							"description": "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low"
						},
						"state_id": {
							"type": "string",
							"description": "Workflow state ID"
						},
						"label_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Label IDs to apply"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment to a Linear issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_id", "body"],
					"properties": {
						"issue_id": {
							"type": "string",
							"description": "The issue ID to comment on"
						},
						"body": {
							"type": "string",
							"description": "Comment body (markdown)"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.create_project",
				Name:        "Create Project",
				Description: "Create a new Linear project",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_ids", "name"],
					"properties": {
						"team_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Team IDs to associate with the project"
						},
						"name": {
							"type": "string",
							"description": "Project name"
						},
						"description": {
							"type": "string",
							"description": "Project description"
						},
						"state": {
							"type": "string",
							"enum": ["planned", "started", "paused", "completed", "cancelled"],
							"description": "Project state"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.search_issues",
				Name:        "Search Issues",
				Description: "Search Linear issues with full-text search or filtered queries",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query (matched against issue titles)"
						},
						"team_id": {
							"type": "string",
							"description": "Filter by team ID"
						},
						"assignee_id": {
							"type": "string",
							"description": "Filter by assignee user ID"
						},
						"state": {
							"type": "string",
							"description": "Filter by workflow state name"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 50,
							"description": "Maximum number of results (default 50, max 100)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "linear", AuthType: "api_key", InstructionsURL: "https://linear.app/docs/graphql/working-with-the-graphql-api#personal-api-keys"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_linear_create_issue_in_team",
				ActionType:  "linear.create_issue",
				Name:        "Create issues in a team",
				Description: "Locks the team and lets the agent choose issue details.",
				Parameters:  json.RawMessage(`{"team_id":"TEAM_ID","title":"*","description":"*"}`),
			},
			{
				ID:          "tpl_linear_search_my_issues",
				ActionType:  "linear.search_issues",
				Name:        "Search my assigned issues",
				Description: "Locks the assignee and lets the agent search freely.",
				Parameters:  json.RawMessage(`{"query":"*","assignee_id":"USER_ID"}`),
			},
			{
				ID:          "tpl_linear_add_comment",
				ActionType:  "linear.add_comment",
				Name:        "Comment on issues",
				Description: "Agent can add comments to any issue.",
				Parameters:  json.RawMessage(`{"issue_id":"*","body":"*"}`),
			},
			{
				ID:          "tpl_linear_update_issue",
				ActionType:  "linear.update_issue",
				Name:        "Update issues",
				Description: "Agent can update any issue's fields.",
				Parameters:  json.RawMessage(`{"issue_id":"*","title":"*","description":"*","priority":"*","state_id":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *LinearConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"linear.create_issue":   &createIssueAction{conn: c},
		"linear.update_issue":   &updateIssueAction{conn: c},
		"linear.add_comment":    &addCommentAction{conn: c},
		"linear.create_project": &createProjectAction{conn: c},
		"linear.search_issues":  &searchIssuesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key. Linear personal API keys are opaque strings with
// no fixed prefix, so we only validate presence.
func (c *LinearConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// graphQLRequest is the standard GraphQL request envelope.
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse is the standard GraphQL response envelope.
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphQLError  `json:"errors,omitempty"`
}

// graphQLError represents a single error in the GraphQL errors array.
type graphQLError struct {
	Message    string                 `json:"message"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// doGraphQL sends a GraphQL request to the Linear API and unmarshals the
// response data into dest. It handles auth, rate limiting, timeouts, and
// maps Linear GraphQL errors to connector error types.
func (c *LinearConnector) doGraphQL(ctx context.Context, creds connectors.Credentials, query string, variables map[string]any, dest any) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}

	gqlReq := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	payload, err := json.Marshal(gqlReq)
	if err != nil {
		return fmt.Errorf("marshaling GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	// Linear uses "Authorization: {api_key}" — no "Bearer" prefix.
	req.Header.Set("Authorization", key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Linear API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Linear API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Linear API request failed: %v", err)}
	}
	defer resp.Body.Close()

	// Handle HTTP-level rate limiting.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Linear API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	// Handle HTTP-level auth errors.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{Message: fmt.Sprintf("Linear API auth error (HTTP %d)", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Linear API response",
		}
	}

	// Check for GraphQL-level errors.
	if len(gqlResp.Errors) > 0 {
		return mapGraphQLErrors(gqlResp.Errors)
	}

	// Unmarshal the data field into the caller's destination.
	if dest != nil && gqlResp.Data != nil {
		if err := json.Unmarshal(gqlResp.Data, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("failed to decode Linear API response data: %v", err),
			}
		}
	}

	return nil
}

// mapGraphQLErrors converts Linear GraphQL errors to the appropriate
// connector error type using the extensions.type field.
func mapGraphQLErrors(errs []graphQLError) error {
	if len(errs) == 0 {
		return nil
	}

	first := errs[0]
	extType := graphQLExtensionType(first)

	switch extType {
	case "authentication_error":
		return &connectors.AuthError{Message: fmt.Sprintf("Linear auth error: %s", first.Message)}
	case "forbidden":
		return &connectors.AuthError{Message: fmt.Sprintf("Linear forbidden: %s", first.Message)}
	case "ratelimited":
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Linear rate limit: %s", first.Message),
			RetryAfter: defaultRetryAfter,
		}
	case "validation_error":
		return &connectors.ValidationError{Message: fmt.Sprintf("Linear validation error: %s", first.Message)}
	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Linear GraphQL error: %s", first.Message),
		}
	}
}

// graphQLExtensionType extracts the "type" field from a GraphQL error's
// extensions map, returning an empty string if not present.
func graphQLExtensionType(e graphQLError) string {
	if e.Extensions == nil {
		return ""
	}
	t, ok := e.Extensions["type"].(string)
	if !ok {
		return ""
	}
	return t
}
