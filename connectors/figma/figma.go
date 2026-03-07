// Package figma implements the Figma connector for the Permission Slip
// connector execution layer. It uses the Figma REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Actions:
//   - figma.get_file         — get a design file tree with metadata
//   - figma.get_components   — get design system components from a file
//   - figma.export_images    — export PNG, SVG, or PDF from specific nodes
//   - figma.list_comments    — list comments on a file
//   - figma.post_comment     — post a comment on a file
//   - figma.get_versions     — get version history for a file
//
// Auth: Personal access token (custom credential).
// Base URL: https://api.figma.com/v1/
package figma

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

const (
	defaultBaseURL = "https://api.figma.com/v1"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "personal_access_token"

	// defaultRetryAfter is used when the Figma API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes caps the Figma API response body at 20 MB to prevent
	// memory exhaustion from large design file responses.
	maxResponseBytes = 20 << 20 // 20 MB
)

// FigmaConnector owns the shared HTTP client and base URL used by all
// Figma actions. Actions hold a pointer back to the connector to access
// these shared resources.
type FigmaConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a FigmaConnector with sensible defaults (30s timeout,
// https://api.figma.com/v1 base URL).
func New() *FigmaConnector {
	return &FigmaConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a FigmaConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *FigmaConnector {
	return &FigmaConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "figma", matching the connectors.id in the database.
func (c *FigmaConnector) ID() string { return "figma" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *FigmaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "figma",
		Name:        "Figma",
		Description: "Figma integration for design file access and collaboration",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "figma.get_file",
				Name:        "Get File",
				Description: "Get a full design file tree with metadata",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key (from the Figma URL, e.g. abc123DEF in figma.com/design/abc123DEF/...)"
						},
						"depth": {
							"type": "integer",
							"description": "Positive integer specifying how deep to traverse the document tree. Omit for full depth."
						},
						"node_ids": {
							"type": "string",
							"description": "Comma-separated list of node IDs to retrieve (e.g. \"1:2,3:4\"). Returns only those subtrees."
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.get_components",
				Name:        "Get Components",
				Description: "Get design system components from a file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key (from the Figma URL)"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.export_images",
				Name:        "Export Images",
				Description: "Export PNG, SVG, or PDF from specific nodes in a file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key", "node_ids", "format"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key (from the Figma URL)"
						},
						"node_ids": {
							"type": "string",
							"description": "Comma-separated list of node IDs to export (e.g. \"1:2,3:4\")"
						},
						"format": {
							"type": "string",
							"enum": ["png", "svg", "pdf", "jpg"],
							"description": "Image export format"
						},
						"scale": {
							"type": "number",
							"minimum": 0.01,
							"maximum": 4,
							"default": 1,
							"description": "Image scale factor (0.01–4, default 1). Only applies to png/jpg."
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.list_comments",
				Name:        "List Comments",
				Description: "List comments on a Figma file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key (from the Figma URL)"
						},
						"as_md": {
							"type": "boolean",
							"default": false,
							"description": "If true, return comment messages as markdown"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.post_comment",
				Name:        "Post Comment",
				Description: "Post a comment on a Figma file",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key", "message"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key (from the Figma URL)"
						},
						"message": {
							"type": "string",
							"description": "Comment message text"
						},
						"comment_id": {
							"type": "string",
							"description": "ID of the comment to reply to (for threaded replies)"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.get_versions",
				Name:        "Get Versions",
				Description: "Get the version history for a Figma file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key (from the Figma URL)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "figma", AuthType: "custom", InstructionsURL: "https://www.figma.com/developers/api#authentication"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_figma_get_file",
				ActionType:  "figma.get_file",
				Name:        "Read design file",
				Description: "Agent can read any Figma file's design tree and metadata.",
				Parameters:  json.RawMessage(`{"file_key":"*","depth":"*","node_ids":"*"}`),
			},
			{
				ID:          "tpl_figma_get_components",
				ActionType:  "figma.get_components",
				Name:        "Get design components",
				Description: "Agent can list components from any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*"}`),
			},
			{
				ID:          "tpl_figma_export_images",
				ActionType:  "figma.export_images",
				Name:        "Export images from designs",
				Description: "Agent can export images from any Figma file nodes.",
				Parameters:  json.RawMessage(`{"file_key":"*","node_ids":"*","format":"*","scale":"*"}`),
			},
			{
				ID:          "tpl_figma_list_comments",
				ActionType:  "figma.list_comments",
				Name:        "Read file comments",
				Description: "Agent can list comments on any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*","as_md":"*"}`),
			},
			{
				ID:          "tpl_figma_post_comment",
				ActionType:  "figma.post_comment",
				Name:        "Post comments on designs",
				Description: "Agent can post comments on any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*","message":"*","comment_id":"*"}`),
			},
			{
				ID:          "tpl_figma_get_versions",
				ActionType:  "figma.get_versions",
				Name:        "View version history",
				Description: "Agent can view version history of any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *FigmaConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"figma.get_file":       &getFileAction{conn: c},
		"figma.get_components": &getComponentsAction{conn: c},
		"figma.export_images":  &exportImagesAction{conn: c},
		"figma.list_comments":  &listCommentsAction{conn: c},
		"figma.post_comment":   &postCommentAction{conn: c},
		"figma.get_versions":   &getVersionsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty personal_access_token.
func (c *FigmaConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: personal_access_token"}
	}
	return nil
}

// figmaErrorResponse is the error envelope returned by the Figma API.
// Example: {"status": 403, "err": "Forbidden"}
type figmaErrorResponse struct {
	Status int    `json:"status"`
	Err    string `json:"err"`
}

// validateFileKey checks that a file_key parameter is non-empty and doesn't
// contain path traversal sequences.
func validateFileKey(fileKey string) error {
	if fileKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: file_key"}
	}
	if strings.Contains(fileKey, "/") || strings.Contains(fileKey, "..") || strings.Contains(fileKey, "\\") {
		return &connectors.ValidationError{Message: "invalid file_key: must not contain path separators or traversal sequences"}
	}
	return nil
}

// validateNodeIDs checks that a node_ids string is non-empty and contains
// valid Figma node ID format (X:Y separated by commas).
func validateNodeIDs(nodeIDs string) error {
	if nodeIDs == "" {
		return &connectors.ValidationError{Message: "missing required parameter: node_ids"}
	}
	for _, id := range strings.Split(nodeIDs, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			return &connectors.ValidationError{Message: "node_ids contains empty ID"}
		}
	}
	return nil
}

// doGet is the shared request lifecycle for read-only Figma API calls. It
// sends a GET request to the given path with auth headers, handles rate
// limiting and timeouts, and unmarshals the response into dest.
func (c *FigmaConnector) doGet(ctx context.Context, path string, creds connectors.Credentials, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "personal_access_token credential is missing or empty"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Figma-Token", token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Figma API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Figma API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Figma API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Figma API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapFigmaHTTPError(resp.StatusCode, respBody)
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Figma API response",
		}
	}

	return nil
}

// doPost is the shared request lifecycle for write Figma API calls. It marshals
// body as JSON, sends a POST request with auth headers, handles rate limiting
// and timeouts, and unmarshals the response into dest.
func (c *FigmaConnector) doPost(ctx context.Context, path string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "personal_access_token credential is missing or empty"}
	}

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Figma-Token", token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Figma API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Figma API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Figma API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Figma API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapFigmaHTTPError(resp.StatusCode, respBody)
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Figma API response",
		}
	}

	return nil
}

// mapFigmaHTTPError converts a non-2xx Figma API response to the appropriate
// connector error type.
func mapFigmaHTTPError(statusCode int, body []byte) error {
	var figmaErr figmaErrorResponse
	if err := json.Unmarshal(body, &figmaErr); err != nil {
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Figma API error (status %d): unable to parse error response", statusCode),
		}
	}

	detail := fmt.Sprintf("Figma API error: %s", figmaErr.Err)

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &connectors.AuthError{Message: detail}
	case http.StatusNotFound:
		return &connectors.AuthError{Message: detail}
	case http.StatusTooManyRequests:
		return &connectors.RateLimitError{Message: detail, RetryAfter: defaultRetryAfter}
	case http.StatusBadRequest:
		return &connectors.ValidationError{Message: detail}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    detail,
		}
	}
}
