// Package make implements the Make (formerly Integromat) connector for the
// Permission Slip connector execution layer. It uses Make's REST API v2 to
// manage and execute automation scenarios.
package make

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

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultRegion  = "us1"
	defaultTimeout = 30 * time.Second

	// credKeyAPIToken is the Make API token credential key.
	credKeyAPIToken = "api_token"

	// credKeyRegion selects the Make data center region.
	credKeyRegion = "region"

	// maxResponseBytes caps the response body at 5 MB.
	maxResponseBytes = 5 << 20

	// tokenPrefix is the expected prefix for Make API tokens.
	tokenPrefix = "Token "
)

// regionBaseURLs maps Make data center regions to their API base URLs.
// This is an allowlist — only these regions are accepted, preventing SSRF.
var regionBaseURLs = map[string]string{
	"us1": "https://us1.make.com/api/v2",
	"us2": "https://us2.make.com/api/v2",
	"eu1": "https://eu1.make.com/api/v2",
	"eu2": "https://eu2.make.com/api/v2",
}

// MakeConnector owns the shared HTTP client used by all Make actions.
type MakeConnector struct {
	client  *http.Client
	baseURL string // overridden only in tests
}

// New creates a MakeConnector with sensible defaults.
func New() *MakeConnector {
	return &MakeConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a MakeConnector that points at a test server,
// bypassing the region allowlist.
func newForTest(client *http.Client, baseURL string) *MakeConnector {
	return &MakeConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "make", matching the connectors.id in the database.
func (c *MakeConnector) ID() string { return "make" }

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *MakeConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "make",
		Name:        "Make",
		Description: "Make (formerly Integromat) integration for workflow automation — manage and run scenarios via the Make REST API. Supports all Make regions (us1, us2, eu1, eu2) via the region credential.",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "make.list_scenarios",
				Name:        "List Scenarios",
				Description: "List automation scenarios for a team",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id"],
					"properties": {
						"team_id": {
							"type": "integer",
							"description": "The team ID whose scenarios to list"
						},
						"limit": {
							"type": "integer",
							"default": 50,
							"description": "Max scenarios to return (1-100)"
						},
						"offset": {
							"type": "integer",
							"default": 0,
							"description": "Number of scenarios to skip for pagination"
						}
					}
				}`)),
			},
			{
				ActionType:  "make.run_scenario",
				Name:        "Run Scenario",
				Description: "Execute a Make scenario. The scenario must be active. Optionally provide input data and wait for the result.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["scenario_id"],
					"properties": {
						"scenario_id": {
							"type": "integer",
							"description": "The ID of the scenario to run"
						},
						"data": {
							"type": "object",
							"description": "Input data for scenario inputs (if the scenario has defined inputs)"
						},
						"responsive": {
							"type": "boolean",
							"default": false,
							"description": "If true, wait for the scenario to finish and return the execution result (up to 40s)"
						}
					}
				}`)),
			},
			{
				ActionType:  "make.get_scenario",
				Name:        "Get Scenario",
				Description: "Get details and current status of a specific scenario",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["scenario_id"],
					"properties": {
						"scenario_id": {
							"type": "integer",
							"description": "The ID of the scenario to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "make.toggle_scenario",
				Name:        "Toggle Scenario",
				Description: "Enable or disable a Make scenario",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["scenario_id", "enabled"],
					"properties": {
						"scenario_id": {
							"type": "integer",
							"description": "The ID of the scenario to toggle"
						},
						"enabled": {
							"type": "boolean",
							"description": "Set to true to enable, false to disable"
						}
					}
				}`)),
			},
			{
				ActionType:  "make.list_executions",
				Name:        "List Executions",
				Description: "List execution history for a specific scenario",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["scenario_id"],
					"properties": {
						"scenario_id": {
							"type": "integer",
							"description": "The scenario ID to list executions for"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Max executions to return (1-100)"
						},
						"offset": {
							"type": "integer",
							"default": 0,
							"description": "Number of executions to skip for pagination"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "make",
				AuthType:        "api_key",
				InstructionsURL: "https://www.make.com/en/help/apps/make/make#connecting-to-make-api",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_make_list_scenarios",
				ActionType:  "make.list_scenarios",
				Name:        "List team scenarios",
				Description: "List all scenarios for a given team.",
				Parameters:  json.RawMessage(`{"team_id":"*","limit":"*","offset":"*"}`),
			},
			{
				ID:          "tpl_make_run_scenario",
				ActionType:  "make.run_scenario",
				Name:        "Run a scenario",
				Description: "Execute a specific scenario with optional input data.",
				Parameters:  json.RawMessage(`{"scenario_id":"*","data":"*","responsive":"*"}`),
			},
			{
				ID:          "tpl_make_get_scenario",
				ActionType:  "make.get_scenario",
				Name:        "Get scenario details",
				Description: "View the details and status of a scenario.",
				Parameters:  json.RawMessage(`{"scenario_id":"*"}`),
			},
			{
				ID:          "tpl_make_toggle_scenario",
				ActionType:  "make.toggle_scenario",
				Name:        "Toggle scenario on/off",
				Description: "Enable or disable a scenario.",
				Parameters:  json.RawMessage(`{"scenario_id":"*","enabled":"*"}`),
			},
			{
				ID:          "tpl_make_list_executions",
				ActionType:  "make.list_executions",
				Name:        "View execution history",
				Description: "List recent executions for a scenario.",
				Parameters:  json.RawMessage(`{"scenario_id":"*","limit":"*","offset":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *MakeConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"make.list_scenarios":  &listScenariosAction{conn: c},
		"make.run_scenario":   &runScenarioAction{conn: c},
		"make.get_scenario":   &getScenarioAction{conn: c},
		"make.toggle_scenario": &toggleScenarioAction{conn: c},
		"make.list_executions": &listExecutionsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain
// a non-empty API token and a valid region if provided.
func (c *MakeConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAPIToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_token"}
	}
	// Validate optional region if provided.
	if region, ok := creds.Get(credKeyRegion); ok && region != "" {
		if _, valid := regionBaseURLs[region]; !valid {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid region %q — must be one of: us1, us2, eu1, eu2", region),
			}
		}
	}
	return nil
}

// getBaseURL returns the base URL for API calls. In production, it uses
// the region credential to select from the allowlist. In tests, it uses
// the connector's baseURL override.
func (c *MakeConnector) getBaseURL(creds connectors.Credentials) string {
	// Test override takes priority.
	if c.baseURL != "" {
		return c.baseURL
	}
	region := defaultRegion
	if r, ok := creds.Get(credKeyRegion); ok && r != "" {
		region = r
	}
	if u, ok := regionBaseURLs[region]; ok {
		return u
	}
	return regionBaseURLs[defaultRegion]
}

// doRequest executes an HTTP request against the Make API and unmarshals the response.
func (c *MakeConnector) doRequest(ctx context.Context, method, path string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyAPIToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "api_token credential is missing or empty"}
	}

	baseURL := c.getBaseURL(creds)
	fullURL := baseURL + path

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", tokenPrefix+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Make API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "Make API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Make API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), 60*time.Second)
		return &connectors.RateLimitError{
			Message:    "Make API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &connectors.AuthError{Message: fmt.Sprintf("Make API auth error (HTTP %d): %s", resp.StatusCode, truncateBody(respBody))}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Make API error (HTTP %d): %s", resp.StatusCode, truncateBody(respBody)),
		}
	}

	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Make API response",
			}
		}
	}

	return nil
}

// maxErrorBodyLen caps error message bodies to avoid leaking large/sensitive
// payloads through error strings.
const maxErrorBodyLen = 512

// truncateBody returns the body truncated to maxErrorBodyLen runes.
func truncateBody(body []byte) string {
	return connectors.TruncateUTF8(string(body), maxErrorBodyLen)
}
