// Package trello implements the Trello connector for the Permission Slip
// connector execution layer. It uses the Trello REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Authentication supports two methods:
//   - Atlassian OAuth 2.0 (3LO): Bearer token in the Authorization header (preferred).
//     Trello is an Atlassian product, so OAuth goes through auth.atlassian.com,
//     not Trello's legacy OAuth 1.0a endpoints.
//   - API key + token: key and token as query parameters (still supported).
//
// The OAuth path takes precedence when both credential types are present.
package trello

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.trello.com/1"
	defaultTimeout = 30 * time.Second

	credKeyAPIKey      = "api_key"
	credKeyToken       = "token"
	credKeyAccessToken = "access_token"

	// defaultRetryAfter is used when Trello returns a rate limit response
	// without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 10 * time.Second

	// maxResponseBytes limits how much data we read from the Trello API
	// to prevent OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// TrelloConnector owns the shared HTTP client and base URL used by all
// Trello actions. Actions hold a pointer back to the connector to access
// these shared resources.
type TrelloConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a TrelloConnector with sensible defaults (30s timeout,
// https://api.trello.com/1 base URL). The HTTP client disables automatic
// redirect following to prevent leaking credential query parameters
// (key/token) to redirect targets.
func New() *TrelloConnector {
	return &TrelloConnector{
		client: &http.Client{
			Timeout:       defaultTimeout,
			CheckRedirect: noRedirect,
		},
		baseURL: defaultBaseURL,
	}
}

// noRedirect prevents the HTTP client from following redirects. When using
// API key auth, credentials are passed as query parameters, and following a
// redirect would forward them to the redirect target. With OAuth Bearer
// tokens, most HTTP clients strip the Authorization header on cross-origin
// redirects, but we block all redirects as a defense-in-depth measure.
func noRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// newForTest creates a TrelloConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *TrelloConnector {
	return &TrelloConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "trello", matching the connectors.id in the database.
func (c *TrelloConnector) ID() string { return "trello" }

// Actions returns the registered action handlers keyed by action_type.
func (c *TrelloConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"trello.create_card":      &createCardAction{conn: c},
		"trello.update_card":      &updateCardAction{conn: c},
		"trello.add_comment":      &addCommentAction{conn: c},
		"trello.move_card":        &moveCardAction{conn: c},
		"trello.create_checklist": &createChecklistAction{conn: c},
		"trello.search_cards":     &searchCardsAction{conn: c},
		"trello.list_boards":      &listBoardsAction{conn: c},
		"trello.create_board":     &createBoardAction{conn: c},
		"trello.list_lists":       &listListsAction{conn: c},
		"trello.create_list":      &createListAction{conn: c},
		"trello.delete_card":      &deleteCardAction{conn: c},
		"trello.add_label":        &addLabelAction{conn: c},
		"trello.list_labels":      &listLabelsAction{conn: c},
		"trello.add_member":       &addMemberAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either a
// non-empty access_token (OAuth) or non-empty api_key + token (API key auth),
// then verifies them against the Trello API by calling GET /1/members/me.
// This catches invalid or revoked credentials early instead of failing on the
// first action.
func (c *TrelloConnector) ValidateCredentials(ctx context.Context, creds connectors.Credentials) error {
	if !hasOAuthCreds(creds) && !hasAPIKeyCreds(creds) {
		return &connectors.ValidationError{
			Message: "missing credentials: provide either an OAuth access_token or an api_key + token pair",
		}
	}

	// Verify credentials by hitting the members/me endpoint.
	var me struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := c.doGet(ctx, creds, "/members/me", map[string]string{"fields": "id,username"}, &me); err != nil {
		return err
	}

	return nil
}

// hasOAuthCreds reports whether the credentials contain a non-empty OAuth access token.
func hasOAuthCreds(creds connectors.Credentials) bool {
	token, ok := creds.Get(credKeyAccessToken)
	return ok && token != ""
}

// hasAPIKeyCreds reports whether the credentials contain non-empty API key and token.
func hasAPIKeyCreds(creds connectors.Credentials) bool {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return false
	}
	token, ok := creds.Get(credKeyToken)
	return ok && token != ""
}

// validateTrelloID checks that a string looks like a valid Trello ID
// (24-character hexadecimal string). This catches common mistakes like
// passing a card URL or name instead of an ID.
func validateTrelloID(value, paramName string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", paramName)}
	}
	if len(value) != 24 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s %q: expected a 24-character Trello ID (e.g. 507f1f77bcf86cd799439011)", paramName, value),
		}
	}
	for _, c := range value {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid %s %q: expected a 24-character hex string — did you pass a URL or name instead of an ID?", paramName, value),
			}
		}
	}
	return nil
}

// authMethod represents how a request should be authenticated.
type authMethod struct {
	// queryParams are appended to the URL (used for API key auth).
	queryParams url.Values
	// bearerToken, if non-empty, is sent as an Authorization: Bearer header (OAuth).
	bearerToken string
}

// resolveAuth determines the auth method from the credentials.
// OAuth (access_token) takes precedence over API key+token when both are present.
func resolveAuth(creds connectors.Credentials) (*authMethod, error) {
	// Prefer OAuth when available.
	if accessToken, ok := creds.Get(credKeyAccessToken); ok && accessToken != "" {
		return &authMethod{bearerToken: accessToken}, nil
	}

	// Fall back to API key + token query parameter auth.
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return nil, &connectors.ValidationError{Message: "missing credentials: provide either an OAuth access_token or an api_key + token pair"}
	}
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "token credential is missing or empty"}
	}
	q := url.Values{}
	q.Set("key", key)
	q.Set("token", token)
	return &authMethod{queryParams: q}, nil
}

// sendAndDecode executes an HTTP request, reads the response, checks for
// errors, and unmarshals the result into respBody. Shared by do() and doGet()
// to eliminate duplicated response-handling logic.
func (c *TrelloConnector) sendAndDecode(req *http.Request, respBody any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Trello API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "Trello API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Trello API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    fmt.Sprintf("failed to decode Trello API response: %v", err),
			}
		}
	}

	return nil
}

// do is the shared request lifecycle for Trello write actions (POST/PUT).
// It authenticates via either Bearer token (OAuth) or query parameters
// (API key+token), marshals reqBody as JSON, and unmarshals the response
// into respBody.
func (c *TrelloConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody any) error {
	auth, err := resolveAuth(creds)
	if err != nil {
		return err
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	if auth.queryParams != nil {
		u.RawQuery = auth.queryParams.Encode()
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+auth.bearerToken)
	}

	return c.sendAndDecode(req, respBody)
}

// doGet is a convenience wrapper for GET requests. It builds query parameters
// from the params map and authenticates via either Bearer token (OAuth) or
// query parameters (API key+token).
func (c *TrelloConnector) doGet(ctx context.Context, creds connectors.Credentials, path string, params map[string]string, respBody any) error {
	auth, err := resolveAuth(creds)
	if err != nil {
		return err
	}

	q := url.Values{}
	if auth.queryParams != nil {
		q = auth.queryParams
	}
	for k, v := range params {
		q.Set(k, v)
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("parsing URL: %w", err)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if auth.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+auth.bearerToken)
	}

	return c.sendAndDecode(req, respBody)
}
