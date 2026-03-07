// Package x implements the X/Twitter connector for the Permission Slip
// connector execution layer. It uses the X API v2 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package x

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
	defaultBaseURL    = "https://api.x.com/2"
	defaultTimeout    = 30 * time.Second
	credKeyToken      = "access_token"
	defaultRetryAfter = 30 * time.Second
)

// XConnector owns the shared HTTP client and base URL used by all
// X/Twitter actions. Actions hold a pointer back to the connector to access
// these shared resources.
type XConnector struct {
	client  *http.Client
	baseURL string
}

// New creates an XConnector with sensible defaults (30s timeout,
// https://api.x.com/2 base URL).
func New() *XConnector {
	return &XConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates an XConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *XConnector {
	return &XConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "x", matching the connectors.id in the database.
func (c *XConnector) ID() string { return "x" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *XConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "x",
		Name:        "X (Twitter)",
		Description: "X/Twitter integration for social media management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "x.post_tweet",
				Name:        "Post Tweet",
				Description: "Post a tweet, reply, or quote tweet",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["text"],
					"properties": {
						"text": {
							"type": "string",
							"maxLength": 280,
							"description": "Tweet text (max 280 characters)"
						},
						"reply_to_tweet_id": {
							"type": "string",
							"description": "Tweet ID to reply to"
						},
						"quote_tweet_id": {
							"type": "string",
							"description": "Tweet ID to quote"
						},
						"media_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Pre-uploaded media IDs to attach"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.delete_tweet",
				Name:        "Delete Tweet",
				Description: "Delete a tweet (irreversible)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tweet_id"],
					"properties": {
						"tweet_id": {
							"type": "string",
							"description": "ID of the tweet to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.send_dm",
				Name:        "Send Direct Message",
				Description: "Send a direct message to a user",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["recipient_id", "text"],
					"properties": {
						"recipient_id": {
							"type": "string",
							"description": "User ID of the recipient"
						},
						"text": {
							"type": "string",
							"maxLength": 10000,
							"description": "Message text (max 10,000 characters)"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.get_user_tweets",
				Name:        "Get User Tweets",
				Description: "Get recent tweets from a specific user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["user_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "User ID to get tweets from"
						},
						"max_results": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 10,
							"description": "Maximum number of results (1-100, default 10)"
						},
						"since_id": {
							"type": "string",
							"description": "Only return tweets after this tweet ID"
						},
						"until_id": {
							"type": "string",
							"description": "Only return tweets before this tweet ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.search_tweets",
				Name:        "Search Tweets",
				Description: "Search recent tweets (7-day window on Basic tier)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query (X search syntax)"
						},
						"max_results": {
							"type": "integer",
							"minimum": 10,
							"maximum": 100,
							"default": 10,
							"description": "Maximum number of results (10-100, default 10)"
						},
						"since_id": {
							"type": "string",
							"description": "Only return tweets after this tweet ID"
						},
						"sort_order": {
							"type": "string",
							"enum": ["recency", "relevancy"],
							"description": "Sort order for results"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.get_me",
				Name:        "Get My Profile",
				Description: "Get the authenticated user's profile info",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "x",
				AuthType:      "oauth2",
				OAuthProvider: "x",
				OAuthScopes:   []string{"tweet.read", "tweet.write", "users.read", "dm.read", "dm.write", "offline.access"},
			},
		},
		OAuthProviders: []connectors.ManifestOAuthProvider{
			{
				ID:           "x",
				AuthorizeURL: "https://x.com/i/oauth2/authorize",
				TokenURL:     "https://api.x.com/2/oauth2/token",
				Scopes:       []string{"tweet.read", "tweet.write", "users.read", "dm.read", "dm.write", "offline.access"},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_x_post_tweet",
				ActionType:  "x.post_tweet",
				Name:        "Post tweets on my behalf",
				Description: "Agent can post tweets with any text content.",
				Parameters:  json.RawMessage(`{"text":"*"}`),
			},
			{
				ID:          "tpl_x_post_reply",
				ActionType:  "x.post_tweet",
				Name:        "Reply to tweets",
				Description: "Agent can reply to any tweet with any text content.",
				Parameters:  json.RawMessage(`{"text":"*","reply_to_tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_delete_tweet",
				ActionType:  "x.delete_tweet",
				Name:        "Delete tweets",
				Description: "Agent can delete any tweet.",
				Parameters:  json.RawMessage(`{"tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_send_dm",
				ActionType:  "x.send_dm",
				Name:        "Send direct messages",
				Description: "Agent can send DMs to any user.",
				Parameters:  json.RawMessage(`{"recipient_id":"*","text":"*"}`),
			},
			{
				ID:          "tpl_x_get_user_tweets",
				ActionType:  "x.get_user_tweets",
				Name:        "Read user tweets",
				Description: "Agent can read tweets from any user.",
				Parameters:  json.RawMessage(`{"user_id":"*"}`),
			},
			{
				ID:          "tpl_x_search_tweets",
				ActionType:  "x.search_tweets",
				Name:        "Search for mentions of my brand",
				Description: "Agent can search tweets with any query.",
				Parameters:  json.RawMessage(`{"query":"*"}`),
			},
			{
				ID:          "tpl_x_get_me",
				ActionType:  "x.get_me",
				Name:        "Read my profile",
				Description: "Agent can read the authenticated user's profile.",
				Parameters:  json.RawMessage(`{}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *XConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"x.post_tweet":      &postTweetAction{conn: c},
		"x.delete_tweet":    &deleteTweetAction{conn: c},
		"x.send_dm":         &sendDMAction{conn: c},
		"x.get_user_tweets": &getUserTweetsAction{conn: c},
		"x.search_tweets":   &searchTweetsAction{conn: c},
		"x.get_me":          &getMeAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token.
func (c *XConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// xErrorResponse is the X API v2 error envelope.
type xErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"errors"`
	Detail string `json:"detail"`
	Title  string `json:"title"`
}

// do is the shared request lifecycle for all X actions. It sends the request
// with Bearer token auth, handles rate limiting and timeouts, and unmarshals
// the response into dest. reqBody may be nil for GET/DELETE requests.
func (c *XConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

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
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("X API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "X API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("X API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if dest != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode X API response",
			}
		}
	}

	return nil
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract X API error message.
	var xErr xErrorResponse
	msg := string(body)
	if json.Unmarshal(body, &xErr) == nil {
		if len(xErr.Errors) > 0 {
			msg = xErr.Errors[0].Message
		} else if xErr.Detail != "" {
			msg = xErr.Detail
		} else if xErr.Title != "" {
			msg = xErr.Title
		}
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("x-rate-limit-reset"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("X API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("X API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("X API permission error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("X API error: %s", msg)}
	}
}
