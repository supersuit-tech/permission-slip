// Package netlify implements the Netlify deployment connector for Permission Slip.
// It uses the Netlify REST API with plain net/http.
package netlify

import (
	_ "embed"
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
	defaultBaseURL = "https://api.netlify.com/api/v1"
	defaultTimeout = 30 * time.Second

	// maxResponseBytes caps the response body we'll read from Netlify APIs
	// to prevent a misbehaving or malicious response from exhausting memory.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// NetlifyConnector owns the shared HTTP client and base URL used by all
// Netlify actions.
type NetlifyConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a NetlifyConnector with sensible defaults.
func New() *NetlifyConnector {
	return &NetlifyConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a NetlifyConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *NetlifyConnector {
	return &NetlifyConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "netlify", matching the connectors.id in the database.
func (c *NetlifyConnector) ID() string { return "netlify" }

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *NetlifyConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "netlify",
		Name:        "Netlify",
		Description: "Netlify deployment management — trigger builds, rollback to previous deploys, check status, and manage environment variables via the Netlify API",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "netlify.list_sites",
				Name:        "List Sites",
				Description: "List all sites in the Netlify account",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"filter": {
							"type": "string",
							"enum": ["all", "owner", "guest"],
							"default": "all",
							"description": "Filter sites by ownership"
						},
						"page": {
							"type": "integer",
							"description": "Page number for pagination",
							"default": 1,
							"minimum": 1
						},
						"per_page": {
							"type": "integer",
							"description": "Number of sites per page (default 20, max 100)",
							"default": 20,
							"minimum": 1,
							"maximum": 100
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.list_deployments",
				Name:        "List Deployments",
				Description: "List deployments for a specific site",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["site_id"],
					"properties": {
						"site_id": {
							"type": "string",
							"description": "The site ID or subdomain"
						},
						"page": {
							"type": "integer",
							"description": "Page number for pagination",
							"default": 1,
							"minimum": 1
						},
						"per_page": {
							"type": "integer",
							"description": "Number of deployments per page (default 20, max 100)",
							"default": 20,
							"minimum": 1,
							"maximum": 100
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.get_deployment",
				Name:        "Get Deployment Status",
				Description: "Get detailed status and information for a specific deployment",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["deploy_id"],
					"properties": {
						"deploy_id": {
							"type": "string",
							"description": "The deploy ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.trigger_deployment",
				Name:        "Trigger Deployment",
				Description: "Trigger a new build and deployment for a site",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["site_id"],
					"properties": {
						"site_id": {
							"type": "string",
							"description": "The site ID or subdomain"
						},
						"branch": {
							"type": "string",
							"description": "Git branch to deploy (defaults to production branch)"
						},
						"clear_cache": {
							"type": "boolean",
							"default": false,
							"description": "Whether to clear the build cache before deploying"
						},
						"title": {
							"type": "string",
							"description": "Optional title for this deploy"
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.rollback_deployment",
				Name:        "Rollback Deployment",
				Description: "Rollback a site to a previous deployment by publishing it",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["site_id", "deploy_id"],
					"properties": {
						"site_id": {
							"type": "string",
							"description": "The site ID or subdomain"
						},
						"deploy_id": {
							"type": "string",
							"description": "The deploy ID to restore/publish"
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.list_env_vars",
				Name:        "List Environment Variables",
				Description: "List all environment variables for a site",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_slug", "site_id"],
					"properties": {
						"account_slug": {
							"type": "string",
							"description": "The account slug (team name)"
						},
						"site_id": {
							"type": "string",
							"description": "The site ID to filter by"
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.set_env_var",
				Name:        "Set Environment Variable",
				Description: "Create or update an environment variable for a site",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_slug", "site_id", "key", "values"],
					"properties": {
						"account_slug": {
							"type": "string",
							"description": "The account slug (team name)"
						},
						"site_id": {
							"type": "string",
							"description": "The site ID to scope the variable to"
						},
						"key": {
							"type": "string",
							"description": "Environment variable name"
						},
						"values": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["value", "context"],
								"properties": {
									"value": {
										"type": "string",
										"description": "Environment variable value"
									},
									"context": {
										"type": "string",
										"enum": ["all", "dev", "branch-deploy", "deploy-preview", "production"],
										"description": "Deploy context for this value"
									}
								}
							},
							"description": "Values per deploy context"
						}
					}
				}`)),
			},
			{
				ActionType:  "netlify.delete_env_var",
				Name:        "Delete Environment Variable",
				Description: "Delete an environment variable from a site",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["account_slug", "key"],
					"properties": {
						"account_slug": {
							"type": "string",
							"description": "The account slug (team name)"
						},
						"key": {
							"type": "string",
							"description": "Environment variable name to delete"
						},
						"site_id": {
							"type": "string",
							"description": "The site ID to scope the deletion"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "netlify", AuthType: "oauth2", OAuthProvider: "netlify"},
			{Service: "netlify-api-key", AuthType: "api_key", InstructionsURL: "https://docs.netlify.com/api/get-started/#authentication"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_netlify_list_sites",
				ActionType:  "netlify.list_sites",
				Name:        "List all sites",
				Description: "Agent can list all sites in the account.",
				Parameters:  json.RawMessage(`{"filter":"*","page":"*","per_page":"*"}`),
			},
			{
				ID:          "tpl_netlify_list_deployments",
				ActionType:  "netlify.list_deployments",
				Name:        "List deployments for any site",
				Description: "Agent can list deployments for any site.",
				Parameters:  json.RawMessage(`{"site_id":"*","page":"*","per_page":"*"}`),
			},
			{
				ID:          "tpl_netlify_get_deployment",
				ActionType:  "netlify.get_deployment",
				Name:        "Check deployment status",
				Description: "Agent can check the status of any deployment.",
				Parameters:  json.RawMessage(`{"deploy_id":"*"}`),
			},
			{
				ID:          "tpl_netlify_trigger_deployment",
				ActionType:  "netlify.trigger_deployment",
				Name:        "Trigger deployment for any site",
				Description: "Agent can trigger new builds and deployments for any site.",
				Parameters:  json.RawMessage(`{"site_id":"*","branch":"*","clear_cache":"*","title":"*"}`),
			},
			{
				ID:          "tpl_netlify_list_env_vars",
				ActionType:  "netlify.list_env_vars",
				Name:        "List environment variables",
				Description: "Agent can view environment variable names and values for any site.",
				Parameters:  json.RawMessage(`{"account_slug":"*","site_id":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *NetlifyConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"netlify.list_sites":          &listSitesAction{conn: c},
		"netlify.list_deployments":    &listDeploymentsAction{conn: c},
		"netlify.get_deployment":      &getDeploymentAction{conn: c},
		"netlify.trigger_deployment":  &triggerDeploymentAction{conn: c},
		"netlify.rollback_deployment": &rollbackDeploymentAction{conn: c},
		"netlify.list_env_vars":       &listEnvVarsAction{conn: c},
		"netlify.set_env_var":         &setEnvVarAction{conn: c},
		"netlify.delete_env_var":      &deleteEnvVarAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either
// a non-empty access_token (OAuth) or api_key. OAuth is preferred when both
// are present.
func (c *NetlifyConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get("access_token"); ok && token != "" {
		return nil
	}
	if key, ok := creds.Get("api_key"); ok && key != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token or api_key"}
}

// getBearerToken returns the bearer token from credentials, preferring
// access_token (OAuth) over api_key.
func getBearerToken(creds connectors.Credentials) string {
	if token, ok := creds.Get("access_token"); ok && token != "" {
		return token
	}
	if key, ok := creds.Get("api_key"); ok && key != "" {
		return key
	}
	return ""
}

// do is the shared request lifecycle for all Netlify actions. It handles:
// - JSON marshaling of request bodies
// - Bearer token authentication (OAuth access_token preferred, API key fallback)
// - Response body size limiting (maxResponseBytes) to prevent memory exhaustion
// - Typed error mapping via checkResponse() (auth, rate limit, validation, timeout)
// - JSON unmarshaling of response bodies into the caller's target struct
func (c *NetlifyConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
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
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token := getBearerToken(creds)
	if token == "" {
		return &connectors.ValidationError{Message: "access_token or api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) || errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Netlify API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Netlify API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Netlify response: %v", err)}
		}
	}
	return nil
}
