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

	// maxResponseBytes caps how much data we read from the X API to prevent
	// OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// XConnector owns the shared HTTP client and base URL used by all
// X/Twitter actions. Actions hold a pointer back to the connector to access
// these shared resources.
type XConnector struct {
	client        *http.Client
	baseURL       string
	uploadBaseURL string // overridden in tests; defaults to uploadBaseURL const
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
		client:        client,
		baseURL:       baseURL,
		uploadBaseURL: baseURL,
	}
}

// ID returns "x", matching the connectors.id in the database.
func (c *XConnector) ID() string { return "x" }

// Actions returns the registered action handlers keyed by action_type.
func (c *XConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"x.post_tweet":      &postTweetAction{conn: c},
		"x.delete_tweet":    &deleteTweetAction{conn: c},
		"x.send_dm":         &sendDMAction{conn: c},
		"x.get_user_tweets": &getUserTweetsAction{conn: c},
		"x.search_tweets":   &searchTweetsAction{conn: c},
		"x.get_me":          &getMeAction{conn: c},
		"x.like_tweet":      &likeTweetAction{conn: c},
		"x.unlike_tweet":    &unlikeTweetAction{conn: c},
		"x.retweet":         &retweetAction{conn: c},
		"x.unretweet":       &unretweetAction{conn: c},
		"x.follow_user":     &followUserAction{conn: c},
		"x.unfollow_user":   &unfollowUserAction{conn: c},
		"x.get_followers":   &getFollowersAction{conn: c},
		"x.get_following":   &getFollowingAction{conn: c},
		"x.upload_media":    &uploadMediaAction{conn: c},
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

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
