// Package hubspot implements the HubSpot connector for the Permission Slip
// connector execution layer. It uses the HubSpot CRM API v3 with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
//
// Auth: HubSpot private app access tokens (API key auth).
// Base URL: https://api.hubapi.com
package hubspot

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
	defaultBaseURL  = "https://api.hubapi.com"
	defaultTimeout  = 30 * time.Second
	maxResponseBody = 10 << 20 // 10 MB — guard against unexpectedly large responses

	// credKeyAPIKey is the credential key for HubSpot private app access tokens.
	// Used in ValidateCredentials and do() — keep in sync.
	credKeyAPIKey = "api_key"
)

// HubSpotConnector owns the shared HTTP client and base URL used by all
// HubSpot actions. Actions hold a pointer back to the connector to access
// these shared resources.
type HubSpotConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a HubSpotConnector with sensible defaults (30s timeout,
// https://api.hubapi.com base URL).
func New() *HubSpotConnector {
	return &HubSpotConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a HubSpotConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *HubSpotConnector {
	return &HubSpotConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "hubspot", matching the connectors.id in the database.
func (c *HubSpotConnector) ID() string { return "hubspot" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *HubSpotConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "hubspot",
		Name:        "HubSpot",
		Description: "HubSpot CRM integration for contacts, deals, tickets, and notes",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "hubspot.create_contact",
				Name:        "Create Contact",
				Description: "Create a new contact in HubSpot CRM",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["email"],
					"properties": {
						"email": {
							"type": "string",
							"description": "Contact email address"
						},
						"firstname": {
							"type": "string",
							"description": "Contact first name"
						},
						"lastname": {
							"type": "string",
							"description": "Contact last name"
						},
						"phone": {
							"type": "string",
							"description": "Contact phone number"
						},
						"company": {
							"type": "string",
							"description": "Contact company name"
						},
						"properties": {
							"type": "object",
							"description": "Additional HubSpot contact properties (property name to value)",
							"additionalProperties": {"type": "string"}
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.update_contact",
				Name:        "Update Contact",
				Description: "Update properties on an existing HubSpot contact",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["contact_id", "properties"],
					"properties": {
						"contact_id": {
							"type": "string",
							"description": "HubSpot contact ID to update"
						},
						"properties": {
							"type": "object",
							"description": "Property name to value map (e.g. {\"email\": \"...\", \"phone\": \"...\"})",
							"additionalProperties": {"type": "string"}
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.create_deal",
				Name:        "Create Deal",
				Description: "Create a new deal in a HubSpot pipeline",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["dealname", "pipeline", "dealstage"],
					"properties": {
						"dealname": {
							"type": "string",
							"description": "Deal name"
						},
						"pipeline": {
							"type": "string",
							"description": "Pipeline ID"
						},
						"dealstage": {
							"type": "string",
							"description": "Deal stage ID"
						},
						"amount": {
							"type": "string",
							"description": "Deal amount"
						},
						"closedate": {
							"type": "string",
							"description": "Expected close date (ISO 8601)"
						},
						"associated_contacts": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Contact IDs to associate with the deal"
						},
						"properties": {
							"type": "object",
							"description": "Additional HubSpot deal properties",
							"additionalProperties": {"type": "string"}
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.create_ticket",
				Name:        "Create Ticket",
				Description: "Create a support ticket in HubSpot",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["subject", "hs_pipeline", "hs_pipeline_stage"],
					"properties": {
						"subject": {
							"type": "string",
							"description": "Ticket subject"
						},
						"content": {
							"type": "string",
							"description": "Ticket body/description"
						},
						"hs_pipeline": {
							"type": "string",
							"description": "Pipeline ID"
						},
						"hs_pipeline_stage": {
							"type": "string",
							"description": "Pipeline stage ID"
						},
						"hs_ticket_priority": {
							"type": "string",
							"description": "Ticket priority (e.g. HIGH, MEDIUM, LOW)"
						},
						"properties": {
							"type": "object",
							"description": "Additional HubSpot ticket properties",
							"additionalProperties": {"type": "string"}
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.add_note",
				Name:        "Add Note",
				Description: "Add an engagement note to a HubSpot CRM record",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["object_type", "object_id", "body"],
					"properties": {
						"object_type": {
							"type": "string",
							"enum": ["contact", "deal", "ticket"],
							"description": "CRM object type to attach the note to"
						},
						"object_id": {
							"type": "string",
							"description": "ID of the CRM object"
						},
						"body": {
							"type": "string",
							"description": "Note content (HTML supported)"
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.search",
				Name:        "Search",
				Description: "Search HubSpot CRM objects with filters",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["object_type", "filters"],
					"properties": {
						"object_type": {
							"type": "string",
							"enum": ["contacts", "deals", "tickets", "companies"],
							"description": "CRM object type to search"
						},
						"filters": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["propertyName", "operator", "value"],
								"properties": {
									"propertyName": {"type": "string", "description": "Property to filter on"},
									"operator": {"type": "string", "description": "Filter operator (EQ, NEQ, LT, LTE, GT, GTE, CONTAINS_TOKEN, etc.)"},
									"value": {"type": "string", "description": "Value to compare against"}
								}
							},
							"description": "Array of filter conditions"
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"description": "Maximum number of results to return (default 10)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "hubspot",
				AuthType:        "api_key",
				InstructionsURL: "https://developers.hubspot.com/docs/api/private-apps",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_hubspot_create_contacts",
				ActionType:  "hubspot.create_contact",
				Name:        "Create contacts",
				Description: "Allow the agent to create new contacts in HubSpot CRM.",
				Parameters:  json.RawMessage(`{"email":"*","firstname":"*","lastname":"*","phone":"*","company":"*"}`),
			},
			{
				ID:          "tpl_hubspot_search_deals",
				ActionType:  "hubspot.search",
				Name:        "Search deals by stage",
				Description: "Search for deals filtered by pipeline stage.",
				Parameters:  json.RawMessage(`{"object_type":"deals","filters":"*"}`),
			},
			{
				ID:          "tpl_hubspot_add_notes",
				ActionType:  "hubspot.add_note",
				Name:        "Log notes on any object",
				Description: "Add engagement notes to contacts, deals, or tickets.",
				Parameters:  json.RawMessage(`{"object_type":"*","object_id":"*","body":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *HubSpotConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"hubspot.create_contact": &createContactAction{conn: c},
		"hubspot.update_contact": &updateContactAction{conn: c},
		"hubspot.create_deal":    &createDealAction{conn: c},
		"hubspot.create_ticket":  &createTicketAction{conn: c},
		"hubspot.add_note":       &addNoteAction{conn: c},
		"hubspot.search":         &searchAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty api_key, which is required for all HubSpot API calls.
func (c *HubSpotConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

// do is the shared request lifecycle for all HubSpot actions. It marshals
// reqBody as JSON, sends the request with auth headers, checks the response
// status, and unmarshals the response into respBody. Either reqBody or
// respBody may be nil.
func (c *HubSpotConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody any) error {
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
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("HubSpot API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("HubSpot API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing HubSpot response: %v", err)}
		}
	}
	return nil
}
