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
		Description: "HubSpot CRM integration for contacts, deals, tickets, notes, marketing, and analytics",
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
			{
				ActionType:  "hubspot.list_deals",
				Name:        "List Deals",
				Description: "Search and list deals in the sales pipeline with optional filtering, sorting, and property selection. Returns dealname, amount, stage, and dates by default.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
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
						"sorts": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["propertyName"],
								"properties": {
									"propertyName": {"type": "string", "description": "Property to sort by"},
									"direction": {"type": "string", "enum": ["ASCENDING", "DESCENDING"], "description": "Sort direction (default ASCENDING)"}
								}
							},
							"description": "Array of sort conditions"
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"description": "Maximum number of results (default 10, max 200)"
						},
						"properties": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Deal properties to include in the response (defaults to dealname, amount, dealstage, pipeline, closedate, createdate, hs_lastmodifieddate)"
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.update_deal_stage",
				Name:        "Update Deal Stage",
				Description: "Move a deal to a different pipeline stage. Use this to advance deals through the sales process (e.g., from qualified to closed-won).",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["deal_id", "pipeline_stage"],
					"properties": {
						"deal_id": {
							"type": "string",
							"description": "HubSpot deal ID to update"
						},
						"pipeline_stage": {
							"type": "string",
							"description": "Target pipeline stage ID"
						},
						"close_date": {
							"type": "string",
							"description": "Updated expected close date (ISO 8601)"
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.enroll_in_workflow",
				Name:        "Enroll in Workflow",
				Description: "Enroll a contact in an automation workflow. Workflows may trigger emails, delays, and branching logic — verify the workflow ID before enrolling.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["flow_id", "contact_id"],
					"properties": {
						"flow_id": {
							"type": "string",
							"description": "Workflow (flow) ID to enroll the contact in"
						},
						"contact_id": {
							"type": "string",
							"description": "Contact ID to enroll"
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.create_email_campaign",
				Name:        "Create Email Campaign",
				Description: "Create a marketing email campaign and optionally send it immediately. When send_now is true, the email is sent to all contacts in the specified lists — use with caution.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "subject", "content"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Internal campaign name"
						},
						"subject": {
							"type": "string",
							"description": "Email subject line"
						},
						"content": {
							"type": "string",
							"description": "Email body content (HTML supported)"
						},
						"list_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Contact list IDs to send to"
						},
						"send_now": {
							"type": "boolean",
							"default": false,
							"description": "If true, send immediately; if false, create as draft"
						}
					}
				}`)),
			},
			{
				ActionType:  "hubspot.get_analytics",
				Name:        "Get Analytics",
				Description: "Get marketing and sales analytics reports with configurable time periods. Use for dashboards, reporting, and performance tracking.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["object_type", "time_period"],
					"properties": {
						"object_type": {
							"type": "string",
							"enum": ["contacts", "deals", "companies", "tickets"],
							"description": "Object type to get analytics for"
						},
						"time_period": {
							"type": "string",
							"enum": ["total", "daily", "weekly", "monthly"],
							"description": "Time period granularity"
						},
						"start": {
							"type": "string",
							"description": "Start date/time (ISO 8601 or epoch milliseconds)"
						},
						"end": {
							"type": "string",
							"description": "End date/time (ISO 8601 or epoch milliseconds)"
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
			{
				ID:          "tpl_hubspot_sales_pipeline",
				ActionType:  "hubspot.update_deal_stage",
				Name:        "Sales pipeline management",
				Description: "Allow the agent to move deals between pipeline stages.",
				Parameters:  json.RawMessage(`{"deal_id":"*","pipeline_stage":"*"}`),
			},
			{
				ID:          "tpl_hubspot_list_deals",
				ActionType:  "hubspot.list_deals",
				Name:        "List and filter deals",
				Description: "Allow the agent to search and list deals with filters.",
				Parameters:  json.RawMessage(`{"filters":"*","sorts":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_hubspot_marketing_readonly",
				ActionType:  "hubspot.get_analytics",
				Name:        "Marketing read-only",
				Description: "Allow the agent to view marketing and sales analytics.",
				Parameters:  json.RawMessage(`{"object_type":"*","time_period":"*","start":"*","end":"*"}`),
			},
			{
				ID:          "tpl_hubspot_workflow_enrollment",
				ActionType:  "hubspot.enroll_in_workflow",
				Name:        "Workflow enrollment",
				Description: "Allow the agent to enroll contacts in automation workflows.",
				Parameters:  json.RawMessage(`{"flow_id":"*","contact_id":"*"}`),
			},
			{
				ID:          "tpl_hubspot_email_drafts",
				ActionType:  "hubspot.create_email_campaign",
				Name:        "Create email drafts",
				Description: "Allow the agent to create draft email campaigns (no sending).",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","content":"*","list_ids":"*","send_now":"false"}`),
			},
			{
				ID:          "tpl_hubspot_full_marketing",
				ActionType:  "hubspot.create_email_campaign",
				Name:        "Full marketing admin",
				Description: "Allow the agent to create and send email campaigns.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","content":"*","list_ids":"*","send_now":"*"}`),
			},
		},
	}
}
