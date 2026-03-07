// Package pagerduty implements the PagerDuty connector for the Permission Slip
// connector execution layer. It uses the PagerDuty REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package pagerduty

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
	defaultBaseURL = "https://api.pagerduty.com"
	defaultTimeout = 30 * time.Second
)

// PagerDutyConnector owns the shared HTTP client and base URL used by all
// PagerDuty actions.
type PagerDutyConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a PagerDutyConnector with sensible defaults.
func New() *PagerDutyConnector {
	return &PagerDutyConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a PagerDutyConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *PagerDutyConnector {
	return &PagerDutyConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "pagerduty", matching the connectors.id in the database.
func (c *PagerDutyConnector) ID() string { return "pagerduty" }

// Manifest returns the connector's metadata manifest.
func (c *PagerDutyConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "pagerduty",
		Name:        "PagerDuty",
		Description: "PagerDuty integration for incident management, alert handling, on-call schedules, and escalations. Supports the full incident lifecycle: triggered → acknowledged → resolved.",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "pagerduty.create_incident",
				Name:        "Create Incident",
				Description: "Create a new incident in PagerDuty, which pages the on-call responder. Use when automated detection identifies an issue that needs human attention.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["service_id", "title"],
					"properties": {
						"service_id": {
							"type": "string",
							"description": "The ID of the PagerDuty service to create the incident on (e.g. PABC123)"
						},
						"title": {
							"type": "string",
							"description": "Incident title — should clearly describe the issue"
						},
						"body": {
							"type": "string",
							"description": "Incident body with additional details, investigation steps, or context"
						},
						"urgency": {
							"type": "string",
							"enum": ["high", "low"],
							"description": "Incident urgency: 'high' pages immediately, 'low' respects notification rules"
						},
						"escalation_policy_id": {
							"type": "string",
							"description": "Override the escalation policy (defaults to the service's policy)"
						}
					}
				}`)),
			},
			{
				ActionType:  "pagerduty.acknowledge_alert",
				Name:        "Acknowledge Alert",
				Description: "Acknowledge an incident to stop further notifications and signal that someone is investigating. Standard triage action — safe for autonomous use on low-severity alerts.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["incident_id"],
					"properties": {
						"incident_id": {
							"type": "string",
							"description": "The ID of the incident to acknowledge (e.g. P1234567)"
						}
					}
				}`)),
			},
			{
				ActionType:  "pagerduty.resolve_incident",
				Name:        "Resolve Incident",
				Description: "Resolve an incident in PagerDuty, marking it as fixed. Only resolve after confirming the underlying issue is actually resolved — premature resolution can mask ongoing problems.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["incident_id"],
					"properties": {
						"incident_id": {
							"type": "string",
							"description": "The ID of the incident to resolve"
						}
					}
				}`)),
			},
			{
				ActionType:  "pagerduty.escalate_incident",
				Name:        "Escalate Incident",
				Description: "Escalate an incident to the next level in the escalation policy, paging additional responders. Use when the current on-call cannot resolve the issue or when urgency increases.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["incident_id", "escalation_level"],
					"properties": {
						"incident_id": {
							"type": "string",
							"description": "The ID of the incident to escalate"
						},
						"escalation_level": {
							"type": "integer",
							"description": "The escalation level to set (2 = first escalation, 3 = second, etc.)"
						},
						"escalation_policy_id": {
							"type": "string",
							"description": "Override the escalation policy for this incident"
						}
					}
				}`)),
			},
			{
				ActionType:  "pagerduty.list_on_call",
				Name:        "List On-Call Schedules",
				Description: "List who is currently on-call, optionally filtered by schedule or escalation policy. Use to identify the right person to escalate to or to provide context during triage.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"schedule_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by schedule IDs"
						},
						"escalation_policy_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by escalation policy IDs"
						},
						"since": {
							"type": "string",
							"description": "Start of the time range (ISO 8601, e.g. 2024-01-15T00:00:00Z)"
						},
						"until": {
							"type": "string",
							"description": "End of the time range (ISO 8601)"
						}
					}
				}`)),
			},
			{
				ActionType:  "pagerduty.add_note",
				Name:        "Add Incident Note",
				Description: "Add a note to an existing incident's timeline. Use to document investigation findings, context gathered from metrics, or actions taken during triage.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["incident_id", "content"],
					"properties": {
						"incident_id": {
							"type": "string",
							"description": "The ID of the incident to add a note to"
						},
						"content": {
							"type": "string",
							"description": "The note content (supports plain text)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "pagerduty", AuthType: "api_key", InstructionsURL: "https://support.pagerduty.com/main/docs/api-access-keys"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_pagerduty_acknowledge_alert",
				ActionType:  "pagerduty.acknowledge_alert",
				Name:        "Acknowledge any alert",
				Description: "Agent can acknowledge any incident. Safe for autonomous triage of low-severity alerts.",
				Parameters:  json.RawMessage(`{"incident_id":"*"}`),
			},
			{
				ID:          "tpl_pagerduty_create_incident",
				ActionType:  "pagerduty.create_incident",
				Name:        "Create incidents",
				Description: "Agent can create incidents on any service.",
				Parameters:  json.RawMessage(`{"service_id":"*","title":"*","body":"*","urgency":"*"}`),
			},
			{
				ID:          "tpl_pagerduty_resolve_incident",
				ActionType:  "pagerduty.resolve_incident",
				Name:        "Resolve any incident",
				Description: "Agent can resolve any incident.",
				Parameters:  json.RawMessage(`{"incident_id":"*"}`),
			},
			{
				ID:          "tpl_pagerduty_escalate_incident",
				ActionType:  "pagerduty.escalate_incident",
				Name:        "Escalate any incident",
				Description: "Agent can escalate any incident to any level.",
				Parameters:  json.RawMessage(`{"incident_id":"*","escalation_level":"*","escalation_policy_id":"*"}`),
			},
			{
				ID:          "tpl_pagerduty_list_on_call",
				ActionType:  "pagerduty.list_on_call",
				Name:        "List on-call schedules",
				Description: "Agent can view on-call schedules.",
				Parameters:  json.RawMessage(`{"schedule_ids":"*","escalation_policy_ids":"*","since":"*","until":"*"}`),
			},
			{
				ID:          "tpl_pagerduty_add_note",
				ActionType:  "pagerduty.add_note",
				Name:        "Add incident notes",
				Description: "Agent can add notes to any incident.",
				Parameters:  json.RawMessage(`{"incident_id":"*","content":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *PagerDutyConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"pagerduty.create_incident":   &createIncidentAction{conn: c},
		"pagerduty.acknowledge_alert": &acknowledgeAlertAction{conn: c},
		"pagerduty.resolve_incident":  &resolveIncidentAction{conn: c},
		"pagerduty.escalate_incident": &escalateIncidentAction{conn: c},
		"pagerduty.list_on_call":      &listOnCallAction{conn: c},
		"pagerduty.add_note":          &addNoteAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key. The optional "email" credential is used as the From
// header on write operations (required by PagerDuty for audit trails).
func (c *PagerDutyConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// do is the shared request lifecycle for all PagerDuty actions.
func (c *PagerDutyConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
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
	req.Header.Set("Authorization", "Token token="+apiKey)

	// PagerDuty requires a From header with a valid user email for write
	// operations. This provides an audit trail of who (or what agent)
	// performed the action.
	if email, ok := creds.Get("email"); ok && email != "" {
		req.Header.Set("From", email)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("PagerDuty API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("PagerDuty API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing PagerDuty response: %v", err)}
		}
	}
	return nil
}
