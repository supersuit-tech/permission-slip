package salesforce

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *SalesforceConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "salesforce",
		Name:        "Salesforce",
		Description: "Salesforce CRM integration for managing records, running queries, and logging activities",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "salesforce.create_record",
				Name:        "Create Record",
				Description: "Create a new record of any sObject type (Lead, Contact, Account, Opportunity, Case, etc.)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sobject_type", "fields"],
					"properties": {
						"sobject_type": {
							"type": "string",
							"description": "Salesforce object type (e.g. Lead, Contact, Account, Opportunity, Case)"
						},
						"fields": {
							"type": "object",
							"description": "Field name to value map (e.g. {\"LastName\": \"Smith\", \"Company\": \"Acme\"})"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.update_record",
				Name:        "Update Record",
				Description: "Update fields on an existing Salesforce record",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sobject_type", "record_id", "fields"],
					"properties": {
						"sobject_type": {
							"type": "string",
							"description": "Salesforce object type (e.g. Lead, Contact, Account, Opportunity, Case)"
						},
						"record_id": {
							"type": "string",
							"description": "The 15 or 18-character Salesforce record ID"
						},
						"fields": {
							"type": "object",
							"description": "Partial field updates — only include fields you want to change"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.query",
				Name:        "Query (SOQL)",
				Description: "Execute a SOQL query to retrieve records from Salesforce",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["soql"],
					"properties": {
						"soql": {
							"type": "string",
							"description": "SOQL query string (e.g. \"SELECT Id, Name FROM Lead WHERE Status = 'Open'\")"
						},
						"max_records": {
							"type": "integer",
							"default": 200,
							"minimum": 1,
							"maximum": 2000,
							"description": "Maximum number of records to return (1-2000, default 200)"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.create_task",
				Name:        "Create Task",
				Description: "Create a task (activity) linked to a record in Salesforce",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["subject"],
					"properties": {
						"subject": {
							"type": "string",
							"description": "Task subject line"
						},
						"what_id": {
							"type": "string",
							"description": "Related record ID (Account, Opportunity, etc.)"
						},
						"who_id": {
							"type": "string",
							"description": "Related Contact or Lead ID"
						},
						"status": {
							"type": "string",
							"default": "Not Started",
							"description": "Task status (default: 'Not Started')"
						},
						"priority": {
							"type": "string",
							"default": "Normal",
							"description": "Task priority (e.g. High, Normal, Low)"
						},
						"due_date": {
							"type": "string",
							"description": "Due date in YYYY-MM-DD format"
						},
						"description": {
							"type": "string",
							"description": "Task description or notes"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.add_note",
				Name:        "Add Note",
				Description: "Add a ContentNote to a Salesforce record",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["parent_id", "title"],
					"properties": {
						"parent_id": {
							"type": "string",
							"description": "The Salesforce record ID to attach the note to"
						},
						"title": {
							"type": "string",
							"description": "Note title"
						},
						"body": {
							"type": "string",
							"description": "Note body content (plain text)"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.create_opportunity",
				Name:        "Create Opportunity",
				Description: "Create a new Salesforce Opportunity with typed fields",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "stage_name", "close_date"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Opportunity name"
						},
						"stage_name": {
							"type": "string",
							"description": "Sales stage (e.g. Prospecting, Qualification, Closed Won)"
						},
						"close_date": {
							"type": "string",
							"description": "Expected close date in YYYY-MM-DD format"
						},
						"amount": {
							"type": "number",
							"description": "Opportunity amount (revenue)"
						},
						"account_id": {
							"type": "string",
							"description": "Related Account record ID (15 or 18 characters)"
						},
						"description": {
							"type": "string",
							"description": "Opportunity description"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.update_opportunity",
				Name:        "Update Opportunity",
				Description: "Update stage, amount, or close date on an existing Opportunity",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["record_id"],
					"properties": {
						"record_id": {
							"type": "string",
							"description": "Opportunity record ID (15 or 18 characters)"
						},
						"stage_name": {
							"type": "string",
							"description": "New sales stage"
						},
						"amount": {
							"type": "number",
							"description": "Updated opportunity amount"
						},
						"close_date": {
							"type": "string",
							"description": "Updated close date in YYYY-MM-DD format"
						},
						"name": {
							"type": "string",
							"description": "Updated opportunity name"
						},
						"description": {
							"type": "string",
							"description": "Updated description"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.create_lead",
				Name:        "Create Lead",
				Description: "Create a new Salesforce Lead with typed fields",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["last_name", "company"],
					"properties": {
						"last_name": {
							"type": "string",
							"description": "Lead last name"
						},
						"company": {
							"type": "string",
							"description": "Lead company name"
						},
						"first_name": {
							"type": "string",
							"description": "Lead first name"
						},
						"email": {
							"type": "string",
							"description": "Lead email address"
						},
						"phone": {
							"type": "string",
							"description": "Lead phone number"
						},
						"title": {
							"type": "string",
							"description": "Lead job title"
						},
						"lead_source": {
							"type": "string",
							"description": "Lead source (e.g. Web, Phone Inquiry, Partner)"
						},
						"status": {
							"type": "string",
							"description": "Lead status (e.g. Open - Not Contacted, Working, Closed - Converted)"
						},
						"website": {
							"type": "string",
							"description": "Lead company website"
						},
						"industry": {
							"type": "string",
							"description": "Lead industry"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.convert_lead",
				Name:        "Convert Lead",
				Description: "Convert a Lead to an Account, Contact, and optionally an Opportunity",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["lead_id", "converted_status"],
					"properties": {
						"lead_id": {
							"type": "string",
							"description": "Lead record ID to convert (15 or 18 characters)"
						},
						"converted_status": {
							"type": "string",
							"description": "Lead status to set after conversion (e.g. Closed - Converted)"
						},
						"account_id": {
							"type": "string",
							"description": "Existing Account ID to merge into (omit to create new account)"
						},
						"contact_id": {
							"type": "string",
							"description": "Existing Contact ID to merge into (omit to create new contact)"
						},
						"opportunity_name": {
							"type": "string",
							"description": "Name for the new Opportunity (ignored if do_not_create_opportunity is true)"
						},
						"do_not_create_opportunity": {
							"type": "boolean",
							"description": "Set to true to skip creating an Opportunity during conversion"
						},
						"owner_id": {
							"type": "string",
							"description": "User ID to assign as owner of the converted records"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.delete_record",
				Name:        "Delete Record",
				Description: "Delete a Salesforce record by ID",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sobject_type", "record_id"],
					"properties": {
						"sobject_type": {
							"type": "string",
							"description": "Salesforce object type (e.g. Lead, Contact, Account, Opportunity)"
						},
						"record_id": {
							"type": "string",
							"description": "The 15 or 18-character Salesforce record ID to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.describe_object",
				Name:        "Describe Object",
				Description: "Get the full schema and metadata for a Salesforce sObject type, including all fields and their types",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sobject_type"],
					"properties": {
						"sobject_type": {
							"type": "string",
							"description": "Salesforce object type to describe (e.g. Lead, Opportunity, Account, or any custom object)"
						}
					}
				}`)),
			},
			{
				ActionType:       "salesforce.list_reports",
				Name:             "List Reports",
				Description:      "List all available Salesforce reports",
				RiskLevel:        "low",
				ParametersSchema: json.RawMessage(`{"type": "object", "properties": {}}`),
			},
			{
				ActionType:  "salesforce.run_report",
				Name:        "Run Report",
				Description: "Execute a Salesforce report and return the results",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["report_id"],
					"properties": {
						"report_id": {
							"type": "string",
							"description": "Report record ID (15 or 18 characters)"
						},
						"include_details": {
							"type": "boolean",
							"description": "Include detailed row-level data in the response (default: false, summary only)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "salesforce",
				AuthType:      "oauth2",
				OAuthProvider: "salesforce",
				OAuthScopes: []string{
					"api",
					"refresh_token",
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_salesforce_create_lead",
				ActionType:  "salesforce.create_record",
				Name:        "Create leads",
				Description: "Agent can create Lead records with any fields.",
				Parameters:  json.RawMessage(`{"sobject_type":"Lead","fields":"*"}`),
			},
			{
				ID:          "tpl_salesforce_create_contact",
				ActionType:  "salesforce.create_record",
				Name:        "Create contacts",
				Description: "Agent can create Contact records with any fields.",
				Parameters:  json.RawMessage(`{"sobject_type":"Contact","fields":"*"}`),
			},
			{
				ID:          "tpl_salesforce_create_any_record",
				ActionType:  "salesforce.create_record",
				Name:        "Create any record type",
				Description: "Agent can create records of any sObject type.",
				Parameters:  json.RawMessage(`{"sobject_type":"*","fields":"*"}`),
			},
			{
				ID:          "tpl_salesforce_update_any_record",
				ActionType:  "salesforce.update_record",
				Name:        "Update any record",
				Description: "Agent can update any record type and field.",
				Parameters:  json.RawMessage(`{"sobject_type":"*","record_id":"*","fields":"*"}`),
			},
			{
				ID:          "tpl_salesforce_query_any",
				ActionType:  "salesforce.query",
				Name:        "Run any SOQL query",
				Description: "Agent can execute any SOQL query against Salesforce.",
				Parameters:  json.RawMessage(`{"soql":"*","max_records":"*"}`),
			},
			{
				ID:          "tpl_salesforce_query_open_opportunities",
				ActionType:  "salesforce.query",
				Name:        "Query open opportunities",
				Description: "Agent can query open opportunities. SOQL is locked to a specific query pattern.",
				Parameters:  json.RawMessage(`{"soql":"SELECT Id, Name, Amount, StageName FROM Opportunity WHERE IsClosed = false","max_records":"*"}`),
			},
			{
				ID:          "tpl_salesforce_create_task_any",
				ActionType:  "salesforce.create_task",
				Name:        "Create tasks on any record",
				Description: "Agent can create tasks linked to any record.",
				Parameters:  json.RawMessage(`{"subject":"*","what_id":"*","who_id":"*","status":"*","priority":"*","due_date":"*","description":"*"}`),
			},
			{
				ID:          "tpl_salesforce_add_note_any",
				ActionType:  "salesforce.add_note",
				Name:        "Add notes to any record",
				Description: "Agent can add notes to any Salesforce record.",
				Parameters:  json.RawMessage(`{"parent_id":"*","title":"*","body":"*"}`),
			},
			{
				ID:          "tpl_salesforce_create_opportunity_any",
				ActionType:  "salesforce.create_opportunity",
				Name:        "Create opportunities",
				Description: "Agent can create Opportunity records with any fields.",
				Parameters:  json.RawMessage(`{"name":"*","stage_name":"*","close_date":"*","amount":"*","account_id":"*","description":"*"}`),
			},
			{
				ID:          "tpl_salesforce_update_opportunity_any",
				ActionType:  "salesforce.update_opportunity",
				Name:        "Update opportunities",
				Description: "Agent can update any Opportunity record fields.",
				Parameters:  json.RawMessage(`{"record_id":"*","stage_name":"*","amount":"*","close_date":"*","name":"*","description":"*"}`),
			},
			{
				ID:          "tpl_salesforce_create_lead_typed",
				ActionType:  "salesforce.create_lead",
				Name:        "Create leads (typed)",
				Description: "Agent can create Lead records using the typed create_lead action.",
				Parameters:  json.RawMessage(`{"last_name":"*","company":"*","first_name":"*","email":"*","phone":"*","title":"*","lead_source":"*","status":"*","website":"*","industry":"*"}`),
			},
			{
				ID:          "tpl_salesforce_describe_object_any",
				ActionType:  "salesforce.describe_object",
				Name:        "Describe any object",
				Description: "Agent can retrieve schema metadata for any Salesforce object type.",
				Parameters:  json.RawMessage(`{"sobject_type":"*"}`),
			},
			{
				ID:          "tpl_salesforce_list_reports",
				ActionType:  "salesforce.list_reports",
				Name:        "List reports",
				Description: "Agent can list available Salesforce reports.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_salesforce_run_report_any",
				ActionType:  "salesforce.run_report",
				Name:        "Run any report",
				Description: "Agent can execute any Salesforce report and retrieve results.",
				Parameters:  json.RawMessage(`{"report_id":"*","include_details":"*"}`),
			},
		},
	}
}
