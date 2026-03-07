// Package datadog implements the Datadog connector for the Permission Slip
// connector execution layer. It uses the Datadog REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package datadog

import (
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
)

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

// Manifest returns the connector's metadata manifest.
func (c *DatadogConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "datadog",
		Name:        "Datadog",
		Description: "Datadog integration for metrics querying, incident management, alert handling, and runbook automation",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "datadog.get_metrics",
				Name:        "Get Metrics",
				Description: "Query time series metrics from Datadog",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query", "from", "to"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Datadog metrics query (e.g. avg:system.cpu.user{host:myhost})"
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
				ActionType:  "datadog.create_incident",
				Name:        "Create Incident",
				Description: "Create a new incident in Datadog",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Incident title"
						},
						"severity": {
							"type": "string",
							"enum": ["SEV-1", "SEV-2", "SEV-3", "SEV-4", "SEV-5", "UNKNOWN"],
							"default": "UNKNOWN",
							"description": "Incident severity level"
						},
						"customer_impact_scope": {
							"type": "string",
							"description": "Description of the customer impact"
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
				Description: "Mute (snooze) a Datadog monitor for a specified duration",
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
							"description": "UNIX epoch timestamp when the mute should end. Omit to mute indefinitely."
						},
						"scope": {
							"type": "string",
							"description": "The scope to apply the mute to (e.g. host:myhost)"
						}
					}
				}`)),
			},
			{
				ActionType:  "datadog.trigger_runbook",
				Name:        "Trigger Runbook",
				Description: "Trigger a Datadog Workflow automation (runbook)",
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
							"description": "Input payload to pass to the workflow"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
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
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *DatadogConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"datadog.get_metrics":    &getMetricsAction{conn: c},
		"datadog.create_incident": &createIncidentAction{conn: c},
		"datadog.snooze_alert":   &snoozeAlertAction{conn: c},
		"datadog.trigger_runbook": &triggerRunbookAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain the
// required api_key and app_key for Datadog API calls.
func (c *DatadogConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	appKey, ok := creds.Get("app_key")
	if !ok || appKey == "" {
		return &connectors.ValidationError{Message: "missing required credential: app_key"}
	}
	return nil
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

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	apiKey, _ := creds.Get("api_key")
	appKey, _ := creds.Get("app_key")
	req.Header.Set("DD-API-KEY", apiKey)
	req.Header.Set("DD-APPLICATION-KEY", appKey)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Datadog API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Datadog API request failed: %v", err)}
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
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Datadog response: %v", err)}
		}
	}
	return nil
}
