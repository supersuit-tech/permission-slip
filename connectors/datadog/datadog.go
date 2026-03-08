// Package datadog implements the Datadog connector for the Permission Slip
// connector execution layer. It uses the Datadog REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package datadog

import (
	_ "embed"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.datadoghq.com"
	defaultTimeout = 30 * time.Second

	// maxResponseBytes caps the API response body at 10 MB to prevent OOM
	// from unexpectedly large responses (e.g., metrics queries returning
	// massive time series data).
	maxResponseBytes = 10 << 20 // 10 MB
)

// OAuthScopes are the OAuth2 scopes requested from Datadog's authorization
// server. Defined here (not in oauth/builtin.go) so the manifest and the
// built-in provider declaration reference the same list — a single source of
// truth that prevents the two from drifting apart.
var OAuthScopes = []string{
	"metrics_read",
	"incidents_read",
	"incidents_write",
	"monitors_read",
	"monitors_write",
	"workflows_run",
}

// siteBaseURLs maps Datadog site identifiers to their API base URLs.
// Users in non-US1 regions must set the "site" credential to route
// requests to the correct Datadog datacenter.
var siteBaseURLs = map[string]string{
	"datadoghq.com":      "https://api.datadoghq.com",
	"us3.datadoghq.com":  "https://api.us3.datadoghq.com",
	"us5.datadoghq.com":  "https://api.us5.datadoghq.com",
	"datadoghq.eu":       "https://api.datadoghq.eu",
	"ap1.datadoghq.com":  "https://api.ap1.datadoghq.com",
	"ddog-gov.com":       "https://api.ddog-gov.com",
}

// DatadogConnector owns the shared HTTP client and base URL used by all
// Datadog actions.
type DatadogConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a DatadogConnector with sensible defaults.
func New() *DatadogConnector {
	return &DatadogConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a DatadogConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *DatadogConnector {
	return &DatadogConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "datadog", matching the connectors.id in the database.
func (c *DatadogConnector) ID() string { return "datadog" }

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *DatadogConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "datadog",
		Name:        "Datadog",
		Description: "Datadog integration for metrics querying, incident management, alert handling, and runbook automation. Supports all Datadog sites (US1, US3, US5, EU, AP1, Gov).",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "datadog.get_metrics",
				Name:        "Get Metrics",
				Description: "Query time series metrics from Datadog using the metrics query language. Use this to gather observability data for triage, capacity planning, or anomaly detection.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query", "from", "to"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Datadog metrics query (e.g. avg:system.cpu.user{host:myhost}). Uses the Datadog query syntax, similar to PromQL."
						},
						"from": {
							"type": "integer",
							"description": "Start of the query window as a UNIX epoch timestamp (seconds)"
						},
						"to": {
							"type": "integer",
							"description": "End of the query window as a UNIX epoch timestamp (seconds)"
						}
					}
				}`)),
			},
			{
				ActionType:  "datadog.get_incident",
				Name:        "Get Incident",
				Description: "Retrieve details of an existing Datadog incident by ID. Use this to gather context during triage before deciding whether to escalate or resolve.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["incident_id"],
					"properties": {
						"incident_id": {
							"type": "string",
							"description": "The ID of the incident to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "datadog.create_incident",
				Name:        "Create Incident",
				Description: "Create a new incident in Datadog. Pages on-call teams and creates a tracking record. Use when automated detection identifies an issue requiring human attention.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Incident title — should clearly describe the issue (e.g. 'High error rate on payments service')"
						},
						"severity": {
							"type": "string",
							"enum": ["SEV-1", "SEV-2", "SEV-3", "SEV-4", "SEV-5", "UNKNOWN"],
							"default": "UNKNOWN",
							"description": "Incident severity: SEV-1 (critical) through SEV-5 (informational)"
						},
						"customer_impact_scope": {
							"type": "string",
							"description": "Description of the customer impact (e.g. '10% of checkout requests failing')"
						},
						"customer_impacted": {
							"type": "boolean",
							"default": false,
							"description": "Whether customers are impacted"
						}
					}
				}`)),
			},
			{
				ActionType:  "datadog.snooze_alert",
				Name:        "Snooze Alert",
				Description: "Mute (snooze) a Datadog monitor for a specified duration. Delays alert notifications — use during planned maintenance or when a known issue is being actively worked.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["monitor_id"],
					"properties": {
						"monitor_id": {
							"type": "integer",
							"description": "The ID of the monitor to mute"
						},
						"end": {
							"type": "integer",
							"description": "UNIX epoch timestamp when the mute should end. Omit to mute indefinitely (not recommended)."
						},
						"scope": {
							"type": "string",
							"description": "Scope to apply the mute to (e.g. host:myhost). Omit to mute all scopes."
						}
					}
				}`)),
			},
			{
				ActionType:  "datadog.trigger_runbook",
				Name:        "Trigger Runbook",
				Description: "Trigger a Datadog Workflow automation (runbook). Executes automated remediation — this is high-risk as workflows can modify infrastructure, restart services, or take other potentially destructive actions.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["workflow_id"],
					"properties": {
						"workflow_id": {
							"type": "string",
							"description": "The ID of the workflow to trigger"
						},
						"payload": {
							"type": "object",
							"description": "Input payload to pass to the workflow (workflow-specific parameters)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "datadog_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "datadog",
				OAuthScopes:   OAuthScopes,
			},
			{Service: "datadog", AuthType: "custom", InstructionsURL: "https://docs.datadoghq.com/account_management/api-app-keys/"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_datadog_get_metrics",
				ActionType:  "datadog.get_metrics",
				Name:        "Query any metrics",
				Description: "Agent can query any Datadog metrics with any time range.",
				Parameters:  json.RawMessage(`{"query":"*","from":"*","to":"*"}`),
			},
			{
				ID:          "tpl_datadog_get_incident",
				ActionType:  "datadog.get_incident",
				Name:        "View any incident",
				Description: "Agent can retrieve details of any Datadog incident.",
				Parameters:  json.RawMessage(`{"incident_id":"*"}`),
			},
			{
				ID:          "tpl_datadog_create_incident",
				ActionType:  "datadog.create_incident",
				Name:        "Create incidents",
				Description: "Agent can create incidents with any severity.",
				Parameters:  json.RawMessage(`{"title":"*","severity":"*","customer_impact_scope":"*","customer_impacted":"*"}`),
			},
			{
				ID:          "tpl_datadog_snooze_alert",
				ActionType:  "datadog.snooze_alert",
				Name:        "Snooze any monitor",
				Description: "Agent can mute any Datadog monitor.",
				Parameters:  json.RawMessage(`{"monitor_id":"*","end":"*","scope":"*"}`),
			},
			{
				ID:          "tpl_datadog_trigger_runbook",
				ActionType:  "datadog.trigger_runbook",
				Name:        "Trigger any runbook",
				Description: "Agent can trigger any Datadog Workflow automation. High-risk: workflows may modify infrastructure.",
				Parameters:  json.RawMessage(`{"workflow_id":"*","payload":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *DatadogConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"datadog.get_metrics":     &getMetricsAction{conn: c},
		"datadog.get_incident":    &getIncidentAction{conn: c},
		"datadog.create_incident": &createIncidentAction{conn: c},
		"datadog.snooze_alert":    &snoozeAlertAction{conn: c},
		"datadog.trigger_runbook": &triggerRunbookAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials are sufficient for
// Datadog API calls. Accepts either:
//   - OAuth: access_token (from the datadog_oauth credential)
//   - Custom: api_key + app_key (from the datadog credential)
//
// If a "site" credential is provided it must be a known Datadog site identifier
// (applies to both auth methods since API calls are region-specific).
func (c *DatadogConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if accessToken, ok := creds.Get("access_token"); ok && accessToken != "" {
		// OAuth path — access_token is sufficient.
		return c.validateSite(creds)
	}
	// Custom auth path — both api_key and app_key are required.
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key or access_token"}
	}
	appKey, ok := creds.Get("app_key")
	if !ok || appKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: app_key"}
	}
	return c.validateSite(creds)
}

// validateSite checks the optional "site" credential when present.
func (c *DatadogConnector) validateSite(creds connectors.Credentials) error {
	if site, ok := creds.Get("site"); ok && site != "" {
		if _, known := siteBaseURLs[site]; !known {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("unknown Datadog site %q — valid sites: datadoghq.com, us3.datadoghq.com, us5.datadoghq.com, datadoghq.eu, ap1.datadoghq.com, ddog-gov.com", site),
			}
		}
	}
	return nil
}

// baseURLForCreds returns the API base URL, respecting the optional "site"
// credential for multi-region support.
func (c *DatadogConnector) baseURLForCreds(creds connectors.Credentials) string {
	if site, ok := creds.Get("site"); ok && site != "" {
		if url, known := siteBaseURLs[site]; known {
			return url
		}
	}
	return c.baseURL
}

// do is the shared request lifecycle for all Datadog actions.
func (c *DatadogConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	baseURL := c.baseURLForCreds(creds)
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Prefer OAuth access_token (Bearer) over api_key + app_key headers.
	if accessToken, ok := creds.Get("access_token"); ok && accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	} else {
		apiKey, _ := creds.Get("api_key")
		appKey, _ := creds.Get("app_key")
		req.Header.Set("DD-API-KEY", apiKey)
		req.Header.Set("DD-APPLICATION-KEY", appKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Datadog API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Datadog API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Datadog response: %v", err)}
		}
	}
	return nil
}
