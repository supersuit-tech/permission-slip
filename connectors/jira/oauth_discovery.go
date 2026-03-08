package jira

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// maxCloudIDCacheSize limits the number of entries in the cloud ID cache
// to prevent unbounded memory growth in multi-tenant deployments. When
// exceeded, expired entries are evicted; if still over limit, the cache
// is cleared entirely.
const maxCloudIDCacheSize = 1000

// cloudIDEntry holds a cached cloud ID with its expiration time.
type cloudIDEntry struct {
	cloudID   string
	expiresAt time.Time
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
		return cloudAPIBaseURL(entry.cloudID), nil
	}

	// Cache miss or expired — fetch from Atlassian.
	cloudID, err := c.fetchCloudID(ctx, accessToken)
	if err != nil {
		return "", err
	}

	// Update cache, evicting stale entries if needed.
	c.cloudIDMu.Lock()
	if len(c.cloudIDCache) >= maxCloudIDCacheSize {
		now := time.Now()
		for k, v := range c.cloudIDCache {
			if now.After(v.expiresAt) {
				delete(c.cloudIDCache, k)
			}
		}
		// If still over limit after evicting expired entries, clear entirely.
		if len(c.cloudIDCache) >= maxCloudIDCacheSize {
			c.cloudIDCache = make(map[string]cloudIDEntry)
		}
	}
	c.cloudIDCache[fp] = cloudIDEntry{
		cloudID:   cloudID,
		expiresAt: time.Now().Add(cloudIDCacheTTL),
	}
	c.cloudIDMu.Unlock()

	return cloudAPIBaseURL(cloudID), nil
}

// cloudAPIBaseURL constructs the Jira Cloud REST API base URL for a given
// cloud ID. The cloud ID is path-escaped to prevent path injection from
// external input.
func cloudAPIBaseURL(cloudID string) string {
	return atlassianCloudAPIBase + "/ex/jira/" + url.PathEscape(cloudID) + "/rest/api/3"
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
		if connectors.IsTimeout(err) || errors.Is(err, context.Canceled) {
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
		return "", classifyResourcesError(resp.StatusCode, resp.Header, body)
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
func classifyResourcesError(statusCode int, header http.Header, body []byte) error {
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
	case http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Atlassian API rate limit exceeded: %s", detail),
			RetryAfter: retryAfter,
		}
	default:
		return &connectors.ExternalError{
			Message: fmt.Sprintf("Atlassian accessible-resources returned status %d: %s", statusCode, detail),
		}
	}
}
