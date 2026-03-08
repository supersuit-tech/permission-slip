// Package notion implements the Notion connector for the Permission Slip
// connector execution layer. It uses the Notion API with plain net/http
// (no third-party SDK) and internal integration tokens (API key auth).
//
// Actions:
//   - notion.create_page   — create pages or database entries
//   - notion.update_page   — update page properties, archive/unarchive
//   - notion.append_blocks — append content blocks to a page
//   - notion.query_database — query a database with filters and sorts
//   - notion.search        — full-text search across shared content
//
// Auth: OAuth 2.0 access token (preferred) or internal integration token (api_key).
// API version: 2022-06-28 (set via Notion-Version header on every request).
package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validatable is implemented by all action parameter structs. It enables the
// shared parseParams helper to unmarshal and validate in one call.
type validatable interface {
	validate() error
}

// parseParams unmarshals JSON parameters and validates them. Every Execute()
// method uses this to reduce boilerplate.
func parseParams(data json.RawMessage, dest validatable) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return dest.validate()
}

// applyPagination sets page_size and start_cursor on the request body.
// A zero page_size defaults to 100.
func applyPagination(body map[string]any, pageSize int, startCursor string) {
	if pageSize == 0 {
		pageSize = 100
	}
	body["page_size"] = pageSize
	if startCursor != "" {
		body["start_cursor"] = startCursor
	}
}

// validateNotionID checks that an ID is safe to embed in a URL path.
// It rejects IDs containing path traversal sequences or URL-unsafe characters.
func validateNotionID(id, fieldName string) error {
	if id == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", fieldName)}
	}
	if strings.Contains(id, "/") || strings.Contains(id, "..") || strings.Contains(id, "\\") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not contain path separators or traversal sequences", fieldName)}
	}
	return nil
}

const (
	defaultBaseURL      = "https://api.notion.com/v1"
	defaultTimeout      = 30 * time.Second
	credKeyAccessToken  = "access_token"
	credKeyAPIKey       = "api_key"
	notionVersion       = "2022-06-28"

	// defaultRetryAfter is used when the Notion API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second
)

// NotionConnector owns the shared HTTP client and base URL used by all
// Notion actions. Actions hold a pointer back to the connector to access
// these shared resources.
type NotionConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a NotionConnector with sensible defaults (30s timeout,
// https://api.notion.com/v1 base URL).
func New() *NotionConnector {
	return &NotionConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a NotionConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *NotionConnector {
	return &NotionConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "notion", matching the connectors.id in the database.
func (c *NotionConnector) ID() string { return "notion" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *NotionConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "notion",
		Name:        "Notion",
		Description: "Notion integration for pages, databases, and content management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "notion.create_page",
				Name:        "Create Page",
				Description: "Create a new page or database entry under a parent page or database",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["parent_id", "title"],
					"properties": {
						"parent_id": {
							"type": "string",
							"description": "Parent page or database ID (UUID, e.g. 8c4d7b3e-a1f2-4e5d-b6c8-9d0e1f2a3b4c)"
						},
						"parent_type": {
							"type": "string",
							"enum": ["page_id", "database_id"],
							"default": "page_id",
							"description": "Whether parent_id is a page or database. Use \"database_id\" when creating database entries."
						},
						"title": {
							"type": "string",
							"description": "Page title text"
						},
						"properties": {
							"type": "object",
							"description": "Database-schema properties object. Overrides the default title-only property when creating database entries. Example: {\"Status\": {\"select\": {\"name\": \"In Progress\"}}}"
						},
						"content": {
							"type": "array",
							"description": "Array of Notion block objects to add as initial page content. Supports paragraph, heading_2, to_do, code, bulleted_list_item, etc.",
							"items": {"type": "object"}
						}
					}
				}`)),
			},
			{
				ActionType:  "notion.update_page",
				Name:        "Update Page",
				Description: "Update properties or archive/unarchive an existing page",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "The page ID to update (UUID, e.g. 8c4d7b3e-a1f2-4e5d-b6c8-9d0e1f2a3b4c)"
						},
						"properties": {
							"type": "object",
							"description": "Partial property updates using Notion property objects. Example: {\"Status\": {\"select\": {\"name\": \"Done\"}}}"
						},
						"archived": {
							"type": "boolean",
							"description": "Set to true to archive (soft-delete) the page, or false to restore it"
						}
					}
				}`)),
			},
			{
				ActionType:  "notion.append_blocks",
				Name:        "Append Content",
				Description: "Append content blocks to the end of a page — ideal for logs, journals, and running notes",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "The page ID to append content to (UUID, e.g. 8c4d7b3e-a1f2-4e5d-b6c8-9d0e1f2a3b4c)"
						},
						"children": {
							"type": "array",
							"description": "Array of Notion block objects (paragraph, heading_2, to_do, code, bulleted_list_item, etc.). Takes precedence over text if both are provided.",
							"items": {"type": "object"}
						},
						"text": {
							"type": "string",
							"description": "Plain text shorthand — auto-wrapped as a paragraph block. Ignored if children is provided. Use children for rich formatting."
						}
					}
				}`)),
			},
			{
				ActionType:  "notion.query_database",
				Name:        "Query Database",
				Description: "Query a Notion database with optional filters, sorts, and pagination",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["database_id"],
					"properties": {
						"database_id": {
							"type": "string",
							"description": "The database ID to query (UUID, e.g. 8c4d7b3e-a1f2-4e5d-b6c8-9d0e1f2a3b4c)"
						},
						"filter": {
							"type": "object",
							"description": "Notion filter object. Example: {\"property\": \"Status\", \"select\": {\"equals\": \"Done\"}}"
						},
						"sorts": {
							"type": "array",
							"description": "Array of sort objects. Example: [{\"property\": \"Created\", \"direction\": \"descending\"}]",
							"items": {"type": "object"}
						},
						"page_size": {
							"type": "integer",
							"default": 100,
							"minimum": 1,
							"maximum": 100,
							"description": "Number of results per page (1–100, default 100)"
						},
						"start_cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response's next_cursor field"
						}
					}
				}`)),
			},
			{
				ActionType:  "notion.search",
				Name:        "Search",
				Description: "Full-text search across all pages and databases shared with the integration",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query text"
						},
						"filter": {
							"type": "object",
							"description": "Filter by object type. Example: {\"property\": \"object\", \"value\": \"page\"} (or \"database\")"
						},
						"page_size": {
							"type": "integer",
							"default": 100,
							"minimum": 1,
							"maximum": 100,
							"description": "Number of results per page (1–100, default 100)"
						},
						"start_cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response's next_cursor field"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "notion_oauth", AuthType: "oauth2", OAuthProvider: "notion"},
			{Service: "notion", AuthType: "api_key", InstructionsURL: "https://developers.notion.com/docs/create-a-notion-integration"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_notion_append_to_page",
				ActionType:  "notion.append_blocks",
				Name:        "Append to daily log page",
				Description: "Agent can append content to a specific page (e.g., daily log, journal).",
				Parameters:  json.RawMessage(`{"page_id":"*","text":"*"}`),
			},
			{
				ID:          "tpl_notion_query_database",
				ActionType:  "notion.query_database",
				Name:        "Query project database",
				Description: "Agent can query a specific database with any filters and sorts.",
				Parameters:  json.RawMessage(`{"database_id":"*","filter":"*","sorts":"*","page_size":"*"}`),
			},
			{
				ID:          "tpl_notion_search",
				ActionType:  "notion.search",
				Name:        "Search all pages",
				Description: "Agent can search across all shared pages and databases.",
				Parameters:  json.RawMessage(`{"query":"*","filter":"*","page_size":"*"}`),
			},
			{
				ID:          "tpl_notion_create_page",
				ActionType:  "notion.create_page",
				Name:        "Create pages freely",
				Description: "Agent can create pages under any parent with any content.",
				Parameters:  json.RawMessage(`{"parent_id":"*","title":"*","properties":"*","content":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *NotionConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"notion.create_page":    &createPageAction{conn: c},
		"notion.update_page":    &updatePageAction{conn: c},
		"notion.append_blocks":  &appendBlocksAction{conn: c},
		"notion.query_database": &queryDatabaseAction{conn: c},
		"notion.search":         &searchAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token (from OAuth) or api_key (internal integration token).
func (c *NotionConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return nil
	}
	if token, ok := creds.Get(credKeyAPIKey); ok && token != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token or api_key"}
}

// resolveToken returns the bearer token from credentials, preferring
// access_token (OAuth) over api_key (internal integration token).
func resolveToken(creds connectors.Credentials) (string, error) {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return token, nil
	}
	if token, ok := creds.Get(credKeyAPIKey); ok && token != "" {
		return token, nil
	}
	return "", &connectors.ValidationError{Message: "access_token or api_key credential is missing or empty"}
}

// notionErrorResponse is the error envelope returned by the Notion API.
// Example: {"object": "error", "status": 401, "code": "unauthorized", "message": "API token is invalid."}
type notionErrorResponse struct {
	Object  string `json:"object"`
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// do is the shared request lifecycle for all Notion actions. It marshals
// body as JSON (if non-nil), sends a request with the given HTTP method to
// the specified path with auth and versioning headers, handles rate limiting
// and timeouts, and unmarshals the response into dest.
func (c *NotionConnector) do(ctx context.Context, httpMethod, path string, creds connectors.Credentials, body any, dest any) error {
	token, err := resolveToken(creds)
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", notionVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Notion API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Notion API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Notion API request failed: %v", err)}
	}
	defer resp.Body.Close()

	// Notion returns 429 for rate limiting with a Retry-After header.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Notion API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	// Check for Notion error responses (non-2xx).
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapNotionHTTPError(resp.StatusCode, respBody)
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Notion API response",
		}
	}

	return nil
}

// mapNotionHTTPError converts a non-2xx Notion API response to the
// appropriate connector error type.
func mapNotionHTTPError(statusCode int, body []byte) error {
	var notionErr notionErrorResponse
	if err := json.Unmarshal(body, &notionErr); err != nil {
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Notion API error (status %d): unable to parse error response", statusCode),
		}
	}

	return mapNotionError(statusCode, notionErr.Code, notionErr.Message)
}

// mapNotionError converts a Notion error code and status to the appropriate
// connector error type. Notion error codes:
// - unauthorized: invalid or expired token
// - restricted_resource: token lacks access to the resource
// - object_not_found: resource doesn't exist or token can't access it
// - validation_error: malformed request
// - rate_limited: too many requests
// - conflict_error: transaction conflict
// - internal_server_error: Notion server error
// - service_unavailable: Notion is down
func mapNotionError(statusCode int, code, message string) error {
	detail := fmt.Sprintf("Notion API error: %s — %s", code, message)

	switch code {
	case "unauthorized":
		return &connectors.AuthError{Message: detail}
	case "restricted_resource", "object_not_found":
		// These can indicate permission issues (token not shared with resource).
		return &connectors.AuthError{Message: detail}
	case "validation_error":
		return &connectors.ValidationError{Message: detail}
	case "rate_limited":
		return &connectors.RateLimitError{Message: detail, RetryAfter: defaultRetryAfter}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    detail,
		}
	}
}
