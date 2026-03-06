// Package shopify implements the Shopify connector for the Permission Slip
// connector execution layer. It uses the Shopify Admin REST API with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package shopify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout = 30 * time.Second
	apiVersion     = "2024-10"

	credKeyAccessToken = "access_token"
	credKeyShopDomain  = "shop_domain"

	// defaultRetryAfter is used when the Shopify API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 2 * time.Second
)

// ShopifyConnector owns the shared HTTP client used by all Shopify actions.
// Unlike GitHub/Slack, the base URL is dynamic — it's derived from the
// shop_domain credential at request time.
type ShopifyConnector struct {
	client     *http.Client
	baseURLFn  func(creds connectors.Credentials) (string, error)
}

// New creates a ShopifyConnector with sensible defaults (30s timeout,
// dynamic base URL from shop_domain credential).
func New() *ShopifyConnector {
	return &ShopifyConnector{
		client:    &http.Client{Timeout: defaultTimeout},
		baseURLFn: shopBaseURL,
	}
}

// newForTest creates a ShopifyConnector that always uses the given base URL,
// ignoring the shop_domain credential. This lets tests point at httptest servers.
func newForTest(client *http.Client, baseURL string) *ShopifyConnector {
	return &ShopifyConnector{
		client:    client,
		baseURLFn: func(_ connectors.Credentials) (string, error) { return baseURL, nil },
	}
}

// shopBaseURL derives the Shopify Admin API base URL from credentials.
// Accepts either the bare subdomain ("mystore") or full domain
// ("mystore.myshopify.com") — anything else is rejected.
func shopBaseURL(creds connectors.Credentials) (string, error) {
	domain, ok := creds.Get(credKeyShopDomain)
	if !ok || domain == "" {
		return "", &connectors.ValidationError{Message: "shop_domain credential is missing or empty"}
	}

	// Strip trailing slash and whitespace.
	domain = strings.TrimRight(strings.TrimSpace(domain), "/")

	// Accept full domain or bare subdomain.
	shop := domain
	if strings.HasSuffix(domain, ".myshopify.com") {
		shop = strings.TrimSuffix(domain, ".myshopify.com")
	} else if strings.Contains(domain, ".") {
		return "", &connectors.ValidationError{
			Message: fmt.Sprintf("shop_domain must be a subdomain (e.g. \"mystore\") or full Shopify domain (e.g. \"mystore.myshopify.com\"), got %q", domain),
		}
	}

	if shop == "" {
		return "", &connectors.ValidationError{Message: "shop_domain resolved to an empty subdomain"}
	}

	return fmt.Sprintf("https://%s.myshopify.com/admin/api/%s", shop, apiVersion), nil
}

// ID returns "shopify", matching the connectors.id in the database.
func (c *ShopifyConnector) ID() string { return "shopify" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup. Phase 2 will add the full action schemas;
// this is a scaffold with no actions registered yet.
func (c *ShopifyConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "shopify",
		Name:        "Shopify",
		Description: "Shopify integration for store management via the Admin REST API",
		Actions:     []connectors.ManifestAction{},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "shopify",
				AuthType:        "api_key",
				InstructionsURL: "https://shopify.dev/docs/apps/build/authentication-authorization/access-tokens/generate-app-access-tokens-admin",
			},
		},
		Templates: []connectors.ManifestTemplate{},
	}
}

// Actions returns the registered action handlers keyed by action_type.
// Phase 2 will add the individual actions here.
func (c *ShopifyConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{}
}

// ValidateCredentials checks that the provided credentials contain both a
// non-empty access_token and a valid shop_domain.
func (c *ShopifyConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	domain, ok := creds.Get(credKeyShopDomain)
	if !ok || domain == "" {
		return &connectors.ValidationError{Message: "missing required credential: shop_domain"}
	}

	// Validate the domain produces a valid base URL.
	if _, err := shopBaseURL(creds); err != nil {
		return err
	}

	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}

	return nil
}

// do is the shared request lifecycle for all Shopify actions. It derives
// the base URL from credentials, marshals reqBody as JSON, sends the request
// with the X-Shopify-Access-Token header, checks the response status, and
// unmarshals the response into respBody.
func (c *ShopifyConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	baseURL, err := c.baseURLFn(creds)
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

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}
	req.Header.Set("X-Shopify-Access-Token", token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Shopify API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Shopify API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Shopify response: %v", err)}
		}
	}
	return nil
}
