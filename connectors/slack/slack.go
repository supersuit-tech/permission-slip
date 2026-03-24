// Package slack implements the Slack connector for the Permission Slip
// connector execution layer. It uses the Slack Web API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL     = "https://slack.com/api"
	defaultTimeout     = 30 * time.Second
	credKeyAccessToken = "access_token"

	// defaultRetryAfter is used when the Slack API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second

	// maxResponseBytes caps the Slack API response body at 10 MB to prevent
	// memory exhaustion from unexpectedly large payloads (e.g., 1000 messages
	// with rich content). Slack's largest documented payloads are well under
	// this limit.
	maxResponseBytes = 10 << 20 // 10 MB
)

// OAuthScopes is the canonical list of user-level OAuth scopes for Slack.
// These are requested via the "user_scope" parameter in the Slack OAuth v2
// authorization URL (no bot "scope" param). The resulting user token (xoxp-)
// is stored as the connection's primary access_token.
var OAuthScopes = []string{
	"channels:history",
	"channels:read",
	"channels:write",
	"channels:write.topic",
	"chat:write",
	"files:read",
	"files:write",
	"groups:history",
	"groups:read",
	"groups:write",
	"im:history",
	"im:read",
	"im:write",
	"mpim:history",
	"mpim:read",
	"mpim:write",
	"reactions:read",
	"reactions:write",
	"search:read",
	"users:read",
	"users:read.email",
}

// SlackConnector owns the shared HTTP client and base URL used by all
// Slack actions. Actions hold a pointer back to the connector to access
// these shared resources.
type SlackConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a SlackConnector with sensible defaults (30s timeout,
// https://slack.com/api base URL).
func New() *SlackConnector {
	return &SlackConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a SlackConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *SlackConnector {
	return &SlackConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "slack", matching the connectors.id in the database.
func (c *SlackConnector) ID() string { return "slack" }

// Actions returns the registered action handlers keyed by action_type.
func (c *SlackConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"slack.send_message":          &sendMessageAction{conn: c},
		"slack.create_channel":        &createChannelAction{conn: c},
		"slack.list_channels":         &listChannelsAction{conn: c},
		"slack.read_channel_messages": &readChannelMessagesAction{conn: c},
		"slack.read_thread":           &readThreadAction{conn: c},
		"slack.schedule_message":      &scheduleMessageAction{conn: c},
		"slack.set_topic":             &setTopicAction{conn: c},
		"slack.invite_to_channel":     &inviteToChannelAction{conn: c},
		"slack.upload_file":           &uploadFileAction{conn: c},
		"slack.add_reaction":          &addReactionAction{conn: c},
		"slack.send_dm":               &sendDMAction{conn: c},
		"slack.update_message":        &updateMessageAction{conn: c},
		"slack.delete_message":        &deleteMessageAction{conn: c},
		"slack.list_users":            &listUsersAction{conn: c},
		"slack.search_messages":       &searchMessagesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a non-empty
// access_token (Slack user OAuth token from the platform).
func (c *SlackConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token"}
}

// slackResponse is the common envelope for Slack Web API responses.
// Every endpoint returns {"ok": true/false, ...}.
type slackResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Needed string `json:"needed,omitempty"` // populated on missing_scope errors
}

// asError converts a failed slackResponse to a typed connector error.
// Prefer this over mapSlackError when the full response is available,
// because it can include the missing scope name for missing_scope errors.
func (r slackResponse) asError() error {
	if r.Error == "missing_scope" && r.Needed != "" {
		return &connectors.AuthError{
			Message: fmt.Sprintf("Slack token is missing the %q OAuth scope — re-authorize the Slack connection to add it", r.Needed),
		}
	}
	return mapSlackError(r.Error)
}

// validatable is implemented by action param structs to validate their fields.
type validatable interface {
	validate() error
}

// parseAndValidate unmarshals JSON parameters into a validatable struct and
// runs its validation. This eliminates the repeated unmarshal + validate
// boilerplate in every action's Execute method.
func parseAndValidate(raw json.RawMessage, params validatable) error {
	if err := json.Unmarshal(raw, params); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return params.validate()
}

// paginationMeta is the shared response_metadata shape for paginated endpoints.
type paginationMeta struct {
	NextCursor string `json:"next_cursor"`
}

// validateChannelID checks that a channel parameter looks like a valid Slack
// channel ID (starts with C, G, or D). This catches common mistakes like
// passing a channel name instead of an ID before hitting the Slack API.
func validateChannelID(channel string) error {
	if channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if len(channel) < 2 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid channel ID %q: expected a Slack channel ID starting with C, G, or D (e.g. C01234567)", channel),
		}
	}
	switch channel[0] {
	case 'C', 'G', 'D':
		return nil
	default:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid channel ID %q: expected a Slack channel ID starting with C, G, or D — did you pass a channel name instead?", channel),
		}
	}
}

// validateUserID checks that a user_id parameter looks like a valid Slack user
// ID (starts with U or W). This catches common mistakes like passing a username
// or email instead of an ID before hitting the Slack API.
func validateUserID(userID string) error {
	if userID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: user_id"}
	}
	if len(userID) < 2 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid user ID %q: expected a Slack user ID starting with U or W (e.g. U01234567)", userID),
		}
	}
	switch userID[0] {
	case 'U', 'W':
		return nil
	default:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid user ID %q: expected a Slack user ID starting with U or W — did you pass a username instead?", userID),
		}
	}
}

// validateMessageTS checks that a message timestamp parameter is non-empty and
// looks like a valid Slack TS value (exactly two numeric parts separated by a
// dot, e.g. "1234567890.123456"). This catches typos and wrong-format values
// before they reach the Slack API.
func validateMessageTS(ts string) error {
	if ts == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ts (message timestamp)"}
	}
	parts := strings.SplitN(ts, ".", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid message timestamp %q: expected format like 1234567890.123456", ts),
		}
	}
	for _, part := range parts {
		for _, c := range part {
			if c < '0' || c > '9' {
				return &connectors.ValidationError{
					Message: fmt.Sprintf("invalid message timestamp %q: expected a numeric Slack timestamp like 1234567890.123456", ts),
				}
			}
		}
	}
	return nil
}

// validateLimit checks that a pagination limit is within the Slack API range (1-1000).
// A zero value means "use default" and is always valid.
func validateLimit(limit int) error {
	if limit != 0 && (limit < 1 || limit > 1000) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 1 and 1000, got %d", limit),
		}
	}
	return nil
}

// getToken extracts the Slack user OAuth access_token from credentials.
func (c *SlackConnector) getToken(creds connectors.Credentials) (string, error) {
	if token, ok := creds.Get(credKeyAccessToken); ok && token != "" {
		return token, nil
	}
	return "", &connectors.ValidationError{Message: "credential is missing: access_token"}
}

// doPost is the shared request lifecycle for all Slack actions. It marshals
// body as JSON, sends a POST to the given Slack API method with auth headers,
// handles rate limiting and timeouts, and unmarshals the response into dest.
// Callers are responsible for checking the Slack-level ok/error fields in dest.
func (c *SlackConnector) doPost(ctx context.Context, method string, creds connectors.Credentials, body any, dest any) error {
	token, err := c.getToken(creds)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Slack API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Slack API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Slack API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	// Handle HTTP-level errors before attempting JSON unmarshal.
	// Slack normally returns 200 with {"ok": false} for app-level errors,
	// but can return non-200 for rate limits, auth failures, and server errors.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return checkHTTPStatus(resp.StatusCode, resp.Header, respBody)
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Slack API response",
		}
	}

	return nil
}

// doGet sends a GET request to a Slack API method with query parameters.
// Used for endpoints like conversations.info and users.info that only accept
// application/x-www-form-urlencoded (not JSON body). The params map is
// encoded as URL query parameters.
func (c *SlackConnector) doGet(ctx context.Context, method string, creds connectors.Credentials, params map[string]string, dest any) error {
	token, err := c.getToken(creds)
	if err != nil {
		return err
	}

	query := neturl.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	url := c.baseURL + "/" + method + "?" + query.Encode()
	return c.doGetURL(ctx, url, token, dest)
}

// doGetURL sends a GET request to the given full URL with Bearer auth and
// unmarshals the JSON response into dest. Used for Slack endpoints that
// require query parameters (e.g., users.lookupByEmail).
func (c *SlackConnector) doGetURL(ctx context.Context, url, token string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Slack API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Slack API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Slack API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return checkHTTPStatus(resp.StatusCode, resp.Header, respBody)
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Slack API response",
		}
	}

	return nil
}

// mapSlackError converts a Slack API error string to the appropriate
// connector error type with user-friendly messages for common errors.
func mapSlackError(slackErr string) error {
	switch slackErr {
	// Auth errors
	case "not_authed", "invalid_auth", "token_revoked", "token_expired", "account_inactive":
		return &connectors.AuthError{Message: fmt.Sprintf("Slack auth error: %s", slackErr)}
	case "missing_scope":
		return &connectors.AuthError{Message: "Slack token is missing a required OAuth scope — re-authorize the Slack connection at https://api.slack.com/apps"}
	case "not_allowed_token_type":
		return &connectors.AuthError{Message: "Slack rejected this request for the token type in use — reconnect Slack"}

	// Rate limiting
	case "ratelimited":
		return &connectors.RateLimitError{Message: "Slack API rate limit exceeded"}

	// Channel errors
	case "channel_not_found":
		return &connectors.ExternalError{StatusCode: 200, Message: "Slack channel not found — verify the channel ID exists and you have access"}
	case "not_in_channel":
		return &connectors.ExternalError{StatusCode: 200, Message: "you are not a member of this channel"}
	case "is_archived":
		return &connectors.ExternalError{StatusCode: 200, Message: "cannot perform this action on an archived channel"}

	// Reaction errors
	case "already_reacted":
		return &connectors.ExternalError{StatusCode: 200, Message: "this emoji reaction has already been added to this message"}
	case "too_many_emoji":
		return &connectors.ExternalError{StatusCode: 200, Message: "too many emoji reactions on this message"}

	// Invite errors
	case "already_in_channel":
		return &connectors.ExternalError{StatusCode: 200, Message: "one or more users are already members of this channel"}
	case "cant_invite_self":
		return &connectors.ExternalError{StatusCode: 200, Message: "you cannot invite yourself to a channel"}
	case "user_not_found":
		return &connectors.ExternalError{StatusCode: 200, Message: "one or more user IDs were not found — verify the user IDs are correct"}

	// Message edit/delete errors
	case "message_not_found":
		return &connectors.ExternalError{StatusCode: 200, Message: "message not found — verify the channel ID and message timestamp are correct"}
	case "cant_delete_message":
		return &connectors.ExternalError{StatusCode: 200, Message: "cannot delete this message — you can only delete your own messages"}
	case "edit_window_closed":
		return &connectors.ExternalError{StatusCode: 200, Message: "the message editing window has closed — messages can only be edited within a limited time"}
	case "cant_update_message":
		return &connectors.ExternalError{StatusCode: 200, Message: "cannot update this message — you can only edit your own messages"}

	// Message errors
	case "time_in_past":
		return &connectors.ExternalError{StatusCode: 200, Message: "scheduled message time is in the past — post_at must be a future Unix timestamp"}
	case "message_too_long":
		return &connectors.ExternalError{StatusCode: 200, Message: "message exceeds Slack's maximum length"}

	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Slack API error: %s", slackErr),
		}
	}
}

// checkHTTPStatus maps non-200 HTTP status codes to typed connector errors.
// Slack normally returns 200 with {"ok": false} for application-level errors,
// but returns standard HTTP status codes for rate limits, auth failures, and
// server errors — especially when the request never reaches the API handler.
func checkHTTPStatus(statusCode int, header http.Header, body []byte) error {
	// Try to extract a Slack error string from the response body.
	var env slackResponse
	msg := ""
	if json.Unmarshal(body, &env) == nil && env.Error != "" {
		msg = env.Error
	}

	switch statusCode {
	case http.StatusUnauthorized:
		if msg == "" {
			msg = "invalid or expired token"
		}
		return &connectors.AuthError{Message: fmt.Sprintf("Slack auth error: %s", msg)}
	case http.StatusForbidden:
		if msg == "" {
			msg = "permission denied"
		}
		return &connectors.AuthError{Message: fmt.Sprintf("Slack permission denied: %s", msg)}
	case http.StatusTooManyRequests:
		// Body message intentionally unused; rate-limit errors only need RetryAfter.
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Slack API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	default:
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", statusCode)
		}
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Slack API error: %s", msg),
		}
	}
}
