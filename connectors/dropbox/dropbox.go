// Package dropbox implements the Dropbox connector for the Permission Slip
// connector execution layer. It uses the Dropbox HTTP API v2 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Dropbox has two API hosts:
//   - api.dropboxapi.com — RPC/metadata endpoints (JSON request/response)
//   - content.dropboxapi.com — content upload/download endpoints
//     (application/octet-stream body, metadata in Dropbox-API-Arg header)
package dropbox

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

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultAPIURL     = "https://api.dropboxapi.com/2"
	defaultContentURL = "https://content.dropboxapi.com/2"
	defaultTimeout    = 30 * time.Second
	credKeyToken      = "access_token"

	// defaultRetryAfter is used when the Dropbox API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes caps the Dropbox API response body at 10 MB to prevent
	// memory exhaustion from unexpectedly large payloads.
	maxResponseBytes = 10 << 20 // 10 MB
)

// OAuthScopes is the canonical list of OAuth scopes required by all Dropbox
// connector actions. This is the single source of truth — referenced by both
// the connector manifest and the built-in OAuth provider registration.
var OAuthScopes = []string{
	"files.content.write",
	"files.content.read",
	"sharing.write",
	"file_requests.read",
}

// DropboxConnector owns the shared HTTP client and base URLs used by all
// Dropbox actions. Actions hold a pointer back to the connector to access
// these shared resources.
type DropboxConnector struct {
	client     *http.Client
	apiURL     string // RPC/metadata endpoints
	contentURL string // content upload/download endpoints
}

// New creates a DropboxConnector with sensible defaults.
func New() *DropboxConnector {
	return &DropboxConnector{
		client:     &http.Client{Timeout: defaultTimeout},
		apiURL:     defaultAPIURL,
		contentURL: defaultContentURL,
	}
}

// newForTest creates a DropboxConnector that points at test servers.
func newForTest(client *http.Client, apiURL, contentURL string) *DropboxConnector {
	return &DropboxConnector{
		client:     client,
		apiURL:     apiURL,
		contentURL: contentURL,
	}
}

// ID returns "dropbox", matching the connectors.id in the database.
func (c *DropboxConnector) ID() string { return "dropbox" }

// Actions returns the registered action handlers keyed by action_type.
func (c *DropboxConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"dropbox.upload_file":   &uploadFileAction{conn: c},
		"dropbox.download_file": &downloadFileAction{conn: c},
		"dropbox.create_folder": &createFolderAction{conn: c},
		"dropbox.share_file":    &shareFileAction{conn: c},
		"dropbox.search":        &searchAction{conn: c},
		"dropbox.move":          &moveAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token. The token is injected by the platform's OAuth
// infrastructure — no OAuth code is needed in this connector.
func (c *DropboxConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get(credKeyToken); ok && token != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token"}
}

// validatable is implemented by action param structs to validate their fields.
type validatable interface {
	validate() error
}

// parseAndValidate unmarshals JSON parameters into a validatable struct and
// runs its validation.
func parseAndValidate(raw json.RawMessage, params validatable) error {
	if err := json.Unmarshal(raw, params); err != nil {
		return &connectors.ValidationError{
			Message: "invalid parameters: check that all parameter types are correct (e.g. strings are quoted, booleans are true/false, numbers are unquoted)",
		}
	}
	return params.validate()
}

// getToken extracts the auth token from credentials.
func (c *DropboxConnector) getToken(creds connectors.Credentials) (string, error) {
	if token, ok := creds.Get(credKeyToken); ok && token != "" {
		return token, nil
	}
	return "", &connectors.ValidationError{Message: "credential is missing: access_token"}
}

// doRPC sends a JSON POST request to a Dropbox RPC/metadata endpoint
// (api.dropboxapi.com). It handles auth, rate limiting, timeouts, and
// unmarshals the response into dest.
func (c *DropboxConnector) doRPC(ctx context.Context, endpoint string, creds connectors.Credentials, body any, dest any) error {
	token, err := c.getToken(creds)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL+"/"+endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	return c.doRequest(req, dest)
}

// doContent sends a content request to a Dropbox content endpoint
// (content.dropboxapi.com). Metadata is passed via the Dropbox-API-Arg header
// as JSON, and the request body contains raw file bytes. The response body is
// returned as raw bytes along with the unmarshaled Dropbox-API-Result header.
func (c *DropboxConnector) doContent(ctx context.Context, endpoint string, creds connectors.Credentials, apiArg any, body io.Reader, resultDest any) ([]byte, error) {
	token, err := c.getToken(creds)
	if err != nil {
		return nil, err
	}

	apiArgJSON, err := json.Marshal(apiArg)
	if err != nil {
		return nil, fmt.Errorf("marshaling Dropbox-API-Arg: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.contentURL+"/"+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Dropbox-API-Arg", string(apiArgJSON))
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, c.mapHTTPError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return nil, &connectors.RateLimitError{
			Message:    "Dropbox API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	// Read up to maxResponseBytes+1 to detect responses that exceed our cap.
	// LimitReader alone would silently truncate, corrupting downloaded content.
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}
	if int64(len(respBody)) > maxResponseBytes {
		return nil, &connectors.ExternalError{
			Message: fmt.Sprintf("Dropbox API response too large (>%d bytes) — for files larger than 10 MB, use chunked download", maxResponseBytes),
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, c.parseErrorResponse(resp.StatusCode, respBody)
	}

	// For download endpoints, metadata is in the Dropbox-API-Result header.
	if resultDest != nil {
		resultHeader := resp.Header.Get("Dropbox-API-Result")
		if resultHeader != "" {
			if err := json.Unmarshal([]byte(resultHeader), resultDest); err != nil {
				return nil, &connectors.ExternalError{Message: "failed to decode Dropbox-API-Result header"}
			}
		}
	}

	return respBody, nil
}

// doContentUpload sends a content upload request and parses the JSON response body.
func (c *DropboxConnector) doContentUpload(ctx context.Context, endpoint string, creds connectors.Credentials, apiArg any, body io.Reader, dest any) error {
	token, err := c.getToken(creds)
	if err != nil {
		return err
	}

	apiArgJSON, err := json.Marshal(apiArg)
	if err != nil {
		return fmt.Errorf("marshaling Dropbox-API-Arg: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.contentURL+"/"+endpoint, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Dropbox-API-Arg", string(apiArgJSON))
	req.Header.Set("Content-Type", "application/octet-stream")

	return c.doRequest(req, dest)
}

// doRequest executes an HTTP request and parses the JSON response.
func (c *DropboxConnector) doRequest(req *http.Request, dest any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return c.mapHTTPError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Dropbox API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseErrorResponse(resp.StatusCode, respBody)
	}

	if dest != nil {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Dropbox API response",
			}
		}
	}

	return nil
}

// mapHTTPError converts net/http transport errors to connector error types.
func (c *DropboxConnector) mapHTTPError(err error) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Dropbox API request timed out: %v", err)}
	}
	if errors.Is(err, context.Canceled) {
		return &connectors.CanceledError{Message: "Dropbox API request canceled"}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Dropbox API request failed: %v", err)}
}

// dropboxError represents the Dropbox API error response format.
type dropboxError struct {
	ErrorSummary string          `json:"error_summary"`
	Error        json.RawMessage `json:"error"`
}

// parseErrorResponse parses a Dropbox error response body and returns the
// appropriate connector error type.
func (c *DropboxConnector) parseErrorResponse(statusCode int, body []byte) error {
	var dbxErr dropboxError
	if err := json.Unmarshal(body, &dbxErr); err != nil {
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Dropbox API error (HTTP %d)", statusCode),
		}
	}

	return mapDropboxError(statusCode, dbxErr.ErrorSummary)
}

// mapDropboxError converts a Dropbox error_summary to the appropriate
// connector error type with user-friendly messages.
func mapDropboxError(statusCode int, errorSummary string) error {
	switch {
	// Auth errors
	case statusCode == 401:
		return &connectors.AuthError{Message: fmt.Sprintf("Dropbox auth error: %s", errorSummary)}
	case strings.Contains(errorSummary, "invalid_access_token"):
		return &connectors.AuthError{Message: "Dropbox access token is invalid or expired"}

	// Path errors
	case strings.Contains(errorSummary, "path/not_found"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "path not found in Dropbox"}
	case strings.Contains(errorSummary, "path/conflict"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "a file or folder already exists at this path"}
	case strings.Contains(errorSummary, "path/malformed_path"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "the path format is invalid — Dropbox paths must start with /"}
	case strings.Contains(errorSummary, "path/disallowed_name"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "the file or folder name is not allowed by Dropbox"}
	case strings.Contains(errorSummary, "path/no_write_permission"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "no write permission for this Dropbox path"}
	case strings.Contains(errorSummary, "path/insufficient_space"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "insufficient Dropbox storage space"}

	// Move-specific errors
	case strings.Contains(errorSummary, "cant_move_folder_into_itself"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "cannot move a folder into itself"}
	case strings.Contains(errorSummary, "cant_move_shared_folder"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "cannot move a shared folder"}

	// Sharing errors
	case strings.Contains(errorSummary, "shared_link_already_exists"):
		return &connectors.ExternalError{StatusCode: statusCode, Message: "a shared link already exists for this file — use the existing link"}

	// Rate limiting
	case strings.Contains(errorSummary, "too_many_requests"):
		return &connectors.RateLimitError{Message: "Dropbox API rate limit exceeded"}

	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Dropbox API error: %s", errorSummary),
		}
	}
}

// validatePath checks that a Dropbox path starts with "/" and is well-formed.
// Dropbox paths are case-insensitive and must not contain double slashes or
// trailing slashes (except root "/").
func validatePath(path, paramName string) error {
	if path == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", paramName)}
	}
	if path[0] != '/' {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s must start with / (e.g. /Documents/report.pdf), got %q", paramName, path),
		}
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s must not end with a trailing slash, got %q", paramName, path),
		}
	}
	if strings.Contains(path, "//") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s contains double slashes — use single slashes (e.g. /Documents/report.pdf), got %q", paramName, path),
		}
	}
	return nil
}
