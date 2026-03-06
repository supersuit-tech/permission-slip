package hubspot

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

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
