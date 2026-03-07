// Package vercel implements the Vercel deployment connector for Permission Slip.
// It uses the Vercel REST API with plain net/http.
package vercel

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
	defaultBaseURL = "https://api.vercel.com"
	defaultTimeout = 30 * time.Second

	// maxResponseBytes caps the response body we'll read from Vercel APIs
	// to prevent a misbehaving or malicious response from exhausting memory.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// VercelConnector owns the shared HTTP client and base URL used by all
// Vercel actions.
type VercelConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a VercelConnector with sensible defaults.
func New() *VercelConnector {
	return &VercelConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a VercelConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *VercelConnector {
	return &VercelConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "vercel", matching the connectors.id in the database.
func (c *VercelConnector) ID() string { return "vercel" }

// Manifest returns the connector's metadata manifest.
func (c *VercelConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "vercel",
		Name:        "Vercel",
		Description: "Vercel deployment management — trigger and promote deployments, rollback, check status, and manage environment variables via the Vercel REST API",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "vercel.list_projects",
				Name:        "List Projects",
				Description: "List all projects in the Vercel account",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request (optional for personal accounts)"
						},
						"limit": {
							"type": "integer",
							"description": "Maximum number of projects to return (default 20, max 100)",
							"default": 20,
							"minimum": 1,
							"maximum": 100
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.list_deployments",
				Name:        "List Deployments",
				Description: "List deployments with optional filtering by project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"project_id": {
							"type": "string",
							"description": "Filter deployments by project ID or name"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						},
						"target": {
							"type": "string",
							"enum": ["production", "preview"],
							"description": "Filter by deployment target"
						},
						"state": {
							"type": "string",
							"enum": ["BUILDING", "ERROR", "INITIALIZING", "QUEUED", "READY", "CANCELED"],
							"description": "Filter by deployment state"
						},
						"limit": {
							"type": "integer",
							"description": "Maximum number of deployments to return (default 20, max 100)",
							"default": 20,
							"minimum": 1,
							"maximum": 100
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.get_deployment",
				Name:        "Get Deployment Status",
				Description: "Get detailed status and information for a specific deployment",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["deployment_id"],
					"properties": {
						"deployment_id": {
							"type": "string",
							"description": "The deployment ID or URL"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.trigger_deployment",
				Name:        "Trigger Deployment",
				Description: "Create a new deployment from a Git branch or commit",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "ref"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID or name"
						},
						"ref": {
							"type": "string",
							"description": "Git ref to deploy (branch name, tag, or commit SHA)"
						},
						"ref_type": {
							"type": "string",
							"enum": ["branch", "commit", "tag"],
							"default": "branch",
							"description": "Type of the Git ref — must match the ref value"
						},
						"target": {
							"type": "string",
							"enum": ["production", "preview"],
							"default": "preview",
							"description": "Deployment target environment"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.rollback_deployment",
				Name:        "Rollback Deployment",
				Description: "Rollback a project to a previous deployment",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "deployment_id"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID or name"
						},
						"deployment_id": {
							"type": "string",
							"description": "The deployment ID to rollback to"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.promote_deployment",
				Name:        "Promote to Production",
				Description: "Promote an existing preview deployment to production. This is the recommended way to ship to production — deploy a preview first, verify it, then promote.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "deployment_id"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID or name"
						},
						"deployment_id": {
							"type": "string",
							"description": "The preview deployment ID to promote to production"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request (required for team projects)"
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.list_env_vars",
				Name:        "List Environment Variables",
				Description: "List all environment variables for a project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID or name"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.set_env_var",
				Name:        "Set Environment Variable",
				Description: "Create or update an environment variable for a project",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "key", "value", "target"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID or name"
						},
						"key": {
							"type": "string",
							"description": "Environment variable name"
						},
						"value": {
							"type": "string",
							"description": "Environment variable value"
						},
						"target": {
							"type": "array",
							"items": {
								"type": "string",
								"enum": ["production", "preview", "development"]
							},
							"description": "Target environments for this variable"
						},
						"type": {
							"type": "string",
							"enum": ["system", "secret", "encrypted", "plain", "sensitive"],
							"default": "encrypted",
							"description": "Type of environment variable"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						}
					}
				}`)),
			},
			{
				ActionType:  "vercel.delete_env_var",
				Name:        "Delete Environment Variable",
				Description: "Delete an environment variable from a project",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "env_id"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID or name"
						},
						"env_id": {
							"type": "string",
							"description": "The environment variable ID to delete"
						},
						"team_id": {
							"type": "string",
							"description": "Team ID to scope the request"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "vercel", AuthType: "api_key", InstructionsURL: "https://vercel.com/docs/rest-api#authentication"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_vercel_list_projects",
				ActionType:  "vercel.list_projects",
				Name:        "List all projects",
				Description: "Agent can list all projects in the account.",
				Parameters:  json.RawMessage(`{"team_id":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_vercel_list_deployments",
				ActionType:  "vercel.list_deployments",
				Name:        "List all deployments",
				Description: "Agent can list deployments across all projects.",
				Parameters:  json.RawMessage(`{"project_id":"*","team_id":"*","target":"*","state":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_vercel_get_deployment",
				ActionType:  "vercel.get_deployment",
				Name:        "Check deployment status",
				Description: "Agent can check the status of any deployment.",
				Parameters:  json.RawMessage(`{"deployment_id":"*","team_id":"*"}`),
			},
			{
				ID:          "tpl_vercel_trigger_preview",
				ActionType:  "vercel.trigger_deployment",
				Name:        "Trigger preview deployments only",
				Description: "Agent can deploy to preview environments but NOT production. Pair with the promote action for a safe deploy-then-promote workflow.",
				Parameters:  json.RawMessage(`{"project_id":"*","ref":"*","target":"preview","team_id":"*"}`),
			},
			{
				ID:          "tpl_vercel_promote_deployment",
				ActionType:  "vercel.promote_deployment",
				Name:        "Promote preview to production",
				Description: "Agent can promote a verified preview deployment to production.",
				Parameters:  json.RawMessage(`{"project_id":"*","deployment_id":"*","team_id":"*"}`),
			},
			{
				ID:          "tpl_vercel_list_env_vars",
				ActionType:  "vercel.list_env_vars",
				Name:        "List environment variables",
				Description: "Agent can view environment variable names and metadata (values are redacted by Vercel for sensitive types).",
				Parameters:  json.RawMessage(`{"project_id":"*","team_id":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *VercelConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"vercel.list_projects":      &listProjectsAction{conn: c},
		"vercel.list_deployments":   &listDeploymentsAction{conn: c},
		"vercel.get_deployment":     &getDeploymentAction{conn: c},
		"vercel.trigger_deployment": &triggerDeploymentAction{conn: c},
		"vercel.promote_deployment":  &promoteDeploymentAction{conn: c},
		"vercel.rollback_deployment": &rollbackDeploymentAction{conn: c},
		"vercel.list_env_vars":      &listEnvVarsAction{conn: c},
		"vercel.set_env_var":        &setEnvVarAction{conn: c},
		"vercel.delete_env_var":     &deleteEnvVarAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key.
func (c *VercelConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// do is the shared request lifecycle for all Vercel actions. It handles:
// - JSON marshaling of request bodies
// - Bearer token authentication via the api_key credential
// - Response body size limiting (maxResponseBytes) to prevent memory exhaustion
// - Typed error mapping via checkResponse() (auth, rate limit, validation, timeout)
// - JSON unmarshaling of response bodies into the caller's target struct
//
// Actions call do() with their specific method, path, and request/response types.
// The path should include any query parameters (use url.Values for building).
func (c *VercelConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
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

	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) || errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Vercel API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Vercel API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Vercel response: %v", err)}
		}
	}
	return nil
}
