// Package jira implements the Jira connector for the Permission Slip
// connector execution layer. It supports two authentication methods:
//   - OAuth 2.0 (Atlassian 3LO) via Bearer token — recommended
//   - Basic auth (email + API token) — alternative
package jira

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// atlassianCloudAPIBase is the base URL for Atlassian Cloud REST APIs
// when using OAuth 2.0 (3LO). The cloud ID identifies the specific
// Jira site and is obtained from the accessible-resources endpoint.
const atlassianCloudAPIBase = "https://api.atlassian.com"

// defaultAccessibleResourcesURL is the Atlassian endpoint that returns
// the list of cloud resources the authenticated user can access. Used to
// discover the cloud ID for constructing Jira API URLs with OAuth.
const defaultAccessibleResourcesURL = atlassianCloudAPIBase + "/oauth/token/accessible-resources"

// cloudIDCacheTTL is how long a discovered cloud ID is cached before
// re-fetching from the accessible-resources endpoint. Cloud IDs are
// stable identifiers, so a long TTL is safe.
const cloudIDCacheTTL = 1 * time.Hour

// validSite matches Atlassian site subdomains: alphanumeric with hyphens.
// Prevents SSRF by ensuring the site value cannot contain path separators,
// fragments, or other characters that would alter the target host.
var validSite = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

const (
	defaultTimeout = 30 * time.Second
	// maxResponseBody is the maximum response body size (10 MB) to prevent OOM
	// from malicious or buggy API responses.
	maxResponseBody = 10 << 20
)

// cloudIDEntry holds a cached cloud ID with its expiration time.
type cloudIDEntry struct {
	cloudID   string
	expiresAt time.Time
}

// JiraConnector owns the shared HTTP client used by all Jira actions.
// The base URL is constructed per-request from the site credential
// (basic auth) or discovered via OAuth cloud ID lookup.
type JiraConnector struct {
	client  *http.Client
	baseURL string // empty for production (derived from credentials); set for tests

	// accessibleResourcesURL is the endpoint used to discover cloud IDs.
	// Defaults to defaultAccessibleResourcesURL; overridden in tests.
	accessibleResourcesURL string

	// cloudIDCache caches discovered cloud IDs keyed by a hash of the
	// access token. This avoids calling the accessible-resources endpoint
	// on every API request. Protected by cloudIDMu.
	cloudIDMu    sync.RWMutex
	cloudIDCache map[string]cloudIDEntry
}

// New creates a JiraConnector with sensible defaults (30s timeout).
func New() *JiraConnector {
	return &JiraConnector{
		client:                 &http.Client{Timeout: defaultTimeout},
		accessibleResourcesURL: defaultAccessibleResourcesURL,
		cloudIDCache:           make(map[string]cloudIDEntry),
	}
}

// newForTest creates a JiraConnector that points at a test server.
// The baseURL overrides all URL construction (both OAuth and basic auth).
func newForTest(client *http.Client, baseURL string) *JiraConnector {
	return &JiraConnector{
		client:                 client,
		baseURL:                baseURL,
		accessibleResourcesURL: defaultAccessibleResourcesURL,
		cloudIDCache:           make(map[string]cloudIDEntry),
	}
}

// newOAuthForTest creates a JiraConnector configured for OAuth testing.
// Unlike newForTest, it does NOT set baseURL so the OAuth cloud ID
// discovery path is exercised. The accessibleResourcesURL is pointed at
// the test server so no external calls are made.
func newOAuthForTest(client *http.Client, resourcesURL string) *JiraConnector {
	return &JiraConnector{
		client:                 client,
		accessibleResourcesURL: resourcesURL,
		cloudIDCache:           make(map[string]cloudIDEntry),
	}
}

// ID returns "jira", matching the connectors.id in the database.
func (c *JiraConnector) ID() string { return "jira" }

// Actions returns the registered action handlers keyed by action_type.
func (c *JiraConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"jira.create_issue":     &createIssueAction{conn: c},
		"jira.update_issue":     &updateIssueAction{conn: c},
		"jira.transition_issue": &transitionIssueAction{conn: c},
		"jira.add_comment":      &addCommentAction{conn: c},
		"jira.assign_issue":     &assignIssueAction{conn: c},
		"jira.search":           &searchAction{conn: c},
	}
}

// isOAuth returns true if the credentials contain an access_token, indicating
// the OAuth 2.0 authentication path should be used.
func isOAuth(creds connectors.Credentials) bool {
	token, ok := creds.Get("access_token")
	return ok && token != ""
}

// ValidateCredentials checks that the provided credentials contain
// the required fields for one of the two supported auth methods:
//   - OAuth: access_token (set automatically from the OAuth connection)
//   - Basic auth: site, email, and api_token
func (c *JiraConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if isOAuth(creds) {
		return nil
	}
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return &connectors.ValidationError{Message: "missing required credential: site"}
	}
	email, ok := creds.Get("email")
	if !ok || email == "" {
		return &connectors.ValidationError{Message: "missing required credential: email"}
	}
	token, ok := creds.Get("api_token")
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	return nil
}

// apiBase returns the base URL for Jira REST API v3 calls.
//
// For OAuth credentials, it fetches the user's accessible resources from
// Atlassian's API to discover the cloud ID, then constructs the URL as
// https://api.atlassian.com/ex/jira/{cloudId}/rest/api/3.
//
// For basic auth, it builds the URL from the site credential as
// https://{site}.atlassian.net/rest/api/3.
//
// In test mode it always returns the test server URL regardless of auth method.
func (c *JiraConnector) apiBase(ctx context.Context, creds connectors.Credentials) (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	if isOAuth(creds) {
		return c.oauthAPIBase(ctx, creds)
	}
	return c.basicAuthAPIBase(creds)
}

// basicAuthAPIBase builds the API base URL from the site credential.
func (c *JiraConnector) basicAuthAPIBase(creds connectors.Credentials) (string, error) {
	site, ok := creds.Get("site")
	if !ok || site == "" {
		return "", &connectors.ValidationError{Message: "missing required credential: site"}
	}
	if !validSite.MatchString(site) {
		return "", &connectors.ValidationError{
			Message: "invalid site credential: must contain only alphanumeric characters and hyphens (e.g. \"my-company\")",
		}
	}
	return "https://" + site + ".atlassian.net/rest/api/3", nil
}

// accessibleResource represents a single Atlassian Cloud resource returned
// by the accessible-resources endpoint.
type accessibleResource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// tokenFingerprint returns a short hash of the access token for use as
// a cache key. We never store raw tokens in the cache map.
func tokenFingerprint(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:8])
}

// oauthAPIBase discovers the Jira cloud ID by calling the Atlassian
// accessible-resources endpoint, then constructs the API base URL.
// Results are cached per access token to avoid redundant API calls.
func (c *JiraConnector) oauthAPIBase(ctx context.Context, creds connectors.Credentials) (string, error) {
	accessToken, _ := creds.Get("access_token")
	fp := tokenFingerprint(accessToken)

	// Check cache first.
	c.cloudIDMu.RLock()
	entry, ok := c.cloudIDCache[fp]
	c.cloudIDMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return atlassianCloudAPIBase + "/ex/jira/" + entry.cloudID + "/rest/api/3", nil
	}

	// Cache miss or expired — fetch from Atlassian.
	cloudID, err := c.fetchCloudID(ctx, accessToken)
	if err != nil {
		return "", err
	}

	// Update cache.
	c.cloudIDMu.Lock()
	c.cloudIDCache[fp] = cloudIDEntry{
		cloudID:   cloudID,
		expiresAt: time.Now().Add(cloudIDCacheTTL),
	}
	c.cloudIDMu.Unlock()

	return atlassianCloudAPIBase + "/ex/jira/" + cloudID + "/rest/api/3", nil
}

// fetchCloudID calls the Atlassian accessible-resources endpoint and
// returns the cloud ID of the first accessible Jira site.
func (c *JiraConnector) fetchCloudID(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.accessibleResourcesURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating accessible-resources request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return "", &connectors.TimeoutError{Message: fmt.Sprintf("accessible-resources request timed out: %v", err)}
		}
		return "", &connectors.ExternalError{Message: fmt.Sprintf("accessible-resources request failed: %v", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("reading accessible-resources response: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return "", classifyResourcesError(resp.StatusCode, body)
	}

	var resources []accessibleResource
	if err := json.Unmarshal(body, &resources); err != nil {
		return "", &connectors.ExternalError{Message: fmt.Sprintf("parsing accessible-resources response: %v", err)}
	}
	if len(resources) == 0 {
		return "", &connectors.ValidationError{
			Message: "no Atlassian Cloud sites found — ensure the OAuth app has access to at least one Jira site",
		}
	}

	// Use the first accessible resource. For users with multiple sites,
	// a future enhancement could let them choose which site to connect.
	cloudID := resources[0].ID
	if cloudID == "" {
		return "", &connectors.ExternalError{Message: "accessible-resources returned a resource with an empty ID"}
	}

	return cloudID, nil
}

// classifyResourcesError maps HTTP error codes from the Atlassian
// accessible-resources endpoint to specific connector error types so
// users get actionable error messages.
func classifyResourcesError(statusCode int, body []byte) error {
	detail := truncate(string(body), 200)
	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{
			Message: "Atlassian OAuth token is invalid or expired — reconnect your Atlassian account",
		}
	case http.StatusForbidden:
		return &connectors.AuthError{
			Message: "Atlassian OAuth app lacks required permissions — check app scopes and re-authorize",
		}
	default:
		return &connectors.ExternalError{
			Message: fmt.Sprintf("Atlassian accessible-resources returned status %d: %s", statusCode, detail),
		}
	}
}

// do is the shared request lifecycle for all Jira actions. It marshals
// reqBody as JSON, sets the appropriate auth header (Bearer for OAuth,
// Basic for legacy), checks the response status, and unmarshals the
// response into respBody. Either reqBody or respBody may be nil.
func (c *JiraConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	base, err := c.apiBase(ctx, creds)
	if err != nil {
		return err
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if isOAuth(creds) {
		accessToken, _ := creds.Get("access_token")
		req.Header.Set("Authorization", "Bearer "+accessToken)
	} else {
		email, ok := creds.Get("email")
		if !ok || email == "" {
			return &connectors.ValidationError{Message: "email credential is missing or empty"}
		}
		token, ok := creds.Get("api_token")
		if !ok || token == "" {
			return &connectors.ValidationError{Message: "api_token credential is missing or empty"}
		}
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(email+":"+token)))
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Jira API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Jira API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Jira response: %v", err)}
		}
	}
	return nil
}
