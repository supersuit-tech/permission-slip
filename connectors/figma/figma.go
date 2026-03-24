// Package figma implements the Figma connector for the Permission Slip
// connector execution layer. It uses the Figma REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Actions:
//   - figma.get_file         — get a design file tree with metadata
//   - figma.get_components   — get design system components from a file
//   - figma.export_images    — export PNG, SVG, PDF, or JPG from specific nodes
//   - figma.list_comments    — list comments on a file
//   - figma.post_comment     — post a comment on a file
//   - figma.get_versions     — get version history for a file
//
// Auth: OAuth2 (primary) or personal access token (fallback).
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
	"net/url"
	"regexp"
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

	// credKeyOAuth is the credential key set by the OAuth flow (access_token).
	credKeyOAuth = "access_token"
	// credKeyPAT is the credential key for personal access tokens (custom auth).
	credKeyPAT = "personal_access_token"
	// credKeyAPIKey is the generic credential key used by the UI's credential
	// dialog (which stores all custom tokens under "api_key").
	credKeyAPIKey = "api_key"

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
		client: &http.Client{
			Timeout:       defaultTimeout,
			CheckRedirect: safeRedirectPolicy(defaultBaseURL),
		},
		baseURL: defaultBaseURL,
	}
}

// safeRedirectPolicy returns an http.Client CheckRedirect function that strips
// authentication headers (X-Figma-Token for PAT, Authorization for OAuth) when
// a redirect goes to a different host than the base URL. This prevents
// credential leakage if the Figma API (or a compromised intermediate) issues a
// cross-origin redirect.
func safeRedirectPolicy(baseURL string) func(*http.Request, []*http.Request) error {
	parsed, _ := url.Parse(baseURL)
	allowedHost := ""
	if parsed != nil {
		allowedHost = parsed.Host
	}
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if req.URL.Host != allowedHost {
			req.Header.Del("X-Figma-Token")
			req.Header.Del("Authorization")
		}
		return nil
	}
}

// newForTest creates a FigmaConnector that points at a test server.
// It applies safeRedirectPolicy to the provided client to match production
// behavior (preventing credential leakage on cross-origin redirects).
func newForTest(client *http.Client, baseURL string) *FigmaConnector {
	client.CheckRedirect = safeRedirectPolicy(baseURL)
	return &FigmaConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "figma", matching the connectors.id in the database.
func (c *FigmaConnector) ID() string { return "figma" }

// Actions returns the registered action handlers keyed by action_type.
func (c *FigmaConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"figma.get_file":       &getFileAction{conn: c},
		"figma.get_components": &getComponentsAction{conn: c},
		"figma.export_images":  &exportImagesAction{conn: c},
		"figma.list_comments":  &listCommentsAction{conn: c},
		"figma.post_comment":   &postCommentAction{conn: c},
		"figma.get_versions":   &getVersionsAction{conn: c},
		"figma.get_styles":     &getStylesAction{conn: c},
		"figma.list_projects":  &listProjectsAction{conn: c},
		"figma.list_files":     &listFilesAction{conn: c},
		"figma.get_variables":  &getVariablesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either
// an OAuth access_token or a personal_access_token.
func (c *FigmaConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	_, _, err := requireToken(creds)
	return err
}

// tokenSource indicates which authentication method is in use.
type tokenSource int

const (
	tokenSourceOAuth tokenSource = iota
	tokenSourcePAT
)

// requireToken extracts a token from credentials. It checks for an OAuth
// access_token first (primary), then falls back to a personal_access_token
// or api_key (PAT stored via the UI). Returns the token value, its source,
// or a ValidationError.
func requireToken(creds connectors.Credentials) (string, tokenSource, error) {
	if token, ok := creds.Get(credKeyOAuth); ok && token != "" {
		return token, tokenSourceOAuth, nil
	}
	if token, ok := creds.Get(credKeyPAT); ok && token != "" {
		return token, tokenSourcePAT, nil
	}
	if token, ok := creds.Get(credKeyAPIKey); ok && token != "" {
		return token, tokenSourcePAT, nil
	}
	return "", 0, &connectors.ValidationError{
		Message: "no Figma credentials found — connect via OAuth or provide a personal access token",
	}
}

// figmaErrorResponse is the error envelope returned by the Figma API.
// Example: {"status": 403, "err": "Forbidden"}
type figmaErrorResponse struct {
	Status int    `json:"status"`
	Err    string `json:"err"`
}

// figmaURLPattern matches Figma file URLs in various formats:
//   - https://www.figma.com/design/FILEKEY/...
//   - https://www.figma.com/file/FILEKEY/...
//   - https://figma.com/design/FILEKEY/...
//   - https://figma.com/file/FILEKEY/...
//
// The file key is captured in the first submatch group.
var figmaURLPattern = regexp.MustCompile(`^https?://(?:www\.)?figma\.com/(?:design|file)/([A-Za-z0-9]+)`)

// nodeIDPattern matches a single Figma node ID in X:Y format where X and Y
// are non-negative integers.
var nodeIDPattern = regexp.MustCompile(`^\d+:\d+$`)

// extractFileKey normalises a file_key parameter: if the value looks like a
// Figma URL it extracts the key portion; otherwise it returns the value as-is.
// This lets callers paste a browser URL directly into the file_key field.
func extractFileKey(raw string) string {
	if m := figmaURLPattern.FindStringSubmatch(raw); len(m) > 1 {
		return m[1]
	}
	return raw
}

// validateFileKey checks that a file_key parameter is non-empty and doesn't
// contain path traversal sequences. It should be called after extractFileKey.
func validateFileKey(fileKey string) error {
	if fileKey == "" {
		return &connectors.ValidationError{Message: "missing required parameter: file_key"}
	}
	if strings.Contains(fileKey, "/") || strings.Contains(fileKey, "..") || strings.Contains(fileKey, "\\") {
		return &connectors.ValidationError{
			Message: "invalid file_key: must not contain path separators or traversal sequences. " +
				"Provide a raw file key (e.g. \"abc123DEF\") or a full Figma URL (e.g. \"https://www.figma.com/design/abc123DEF/...\").",
		}
	}
	return nil
}

// validateNodeIDs checks that a node_ids string is non-empty and each ID
// matches the Figma node ID format (X:Y where X and Y are integers).
func validateNodeIDs(nodeIDs string) error {
	if nodeIDs == "" {
		return &connectors.ValidationError{Message: "missing required parameter: node_ids"}
	}
	for _, id := range strings.Split(nodeIDs, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			return &connectors.ValidationError{Message: "node_ids contains empty ID"}
		}
		if !nodeIDPattern.MatchString(id) {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid node ID %q: must be in X:Y format (e.g. \"1:2\")", id),
			}
		}
	}
	return nil
}

// setAuthHeader sets the appropriate authentication header on the request
// based on the token source. OAuth tokens use the standard Authorization
// header; personal access tokens use the Figma-specific X-Figma-Token header.
func setAuthHeader(req *http.Request, token string, src tokenSource) {
	switch src {
	case tokenSourceOAuth:
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		req.Header.Set("X-Figma-Token", token)
	}
}

// doGet is the shared request lifecycle for read-only Figma API calls.
func (c *FigmaConnector) doGet(ctx context.Context, path string, creds connectors.Credentials, dest any) error {
	token, src, err := requireToken(creds)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	setAuthHeader(req, token, src)

	return c.doRequest(req, dest)
}

// doPost is the shared request lifecycle for write Figma API calls.
func (c *FigmaConnector) doPost(ctx context.Context, path string, creds connectors.Credentials, body any, dest any) error {
	token, src, err := requireToken(creds)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	setAuthHeader(req, token, src)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.doRequest(req, dest)
}

// doRequest executes an HTTP request and handles the Figma API response
// lifecycle: timeouts, rate limiting, error mapping, and JSON decoding.
// Shared by doGet and doPost to eliminate duplicated response handling.
func (c *FigmaConnector) doRequest(req *http.Request, dest any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Figma API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "Figma API request canceled"}
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
		return &connectors.AuthError{Message: detail + " — check that your OAuth connection or personal access token is valid and has access to this resource"}
	case http.StatusNotFound:
		// 404 indicates the requested resource (file, node, etc.) was not found
		// or is invalid. Mapped to ValidationError for consistency with other
		// connectors (e.g. zoom).
		return &connectors.ValidationError{Message: detail + " — the file key may be incorrect, or the resource does not exist"}
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
