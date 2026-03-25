package salesforce

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//
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
							"description": "Salesforce object type (e.g. Lead, Contact, Account, Opportunity, Case)",
							"x-ui": {"label": "Object type", "placeholder": "Lead", "help_text": "Salesforce object API name — e.g. Lead, Contact, Account, Opportunity"}
						},
						"fields": {
							"type": "object",
							"description": "Field name to value map (e.g. {\"LastName\": \"Smith\", \"Company\": \"Acme\"})",
							"x-ui": {"label": "Fields", "help_text": "Key-value object of field API names to values"}
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
							"description": "Salesforce object type (e.g. Lead, Contact, Account, Opportunity, Case)",
							"x-ui": {"label": "Object type", "placeholder": "Lead", "help_text": "Salesforce object API name — e.g. Lead, Contact, Account, Opportunity"}
						},
						"record_id": {
							"type": "string",
							"description": "The 15 or 18-character Salesforce record ID",
							"x-ui": {"label": "Record ID", "placeholder": "001xx000003DGbYAAW", "help_text": "15 or 18-character Salesforce record ID"}
						},
						"fields": {
							"type": "object",
							"description": "Partial field updates — only include fields you want to change",
							"x-ui": {"label": "Fields", "help_text": "Key-value object of field API names to values"}
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
							"description": "SOQL query string (e.g. \"SELECT Id, Name FROM Lead WHERE Status = 'Open'\")",
							"x-ui": {"label": "SOQL query", "widget": "textarea", "placeholder": "SELECT Id, Name FROM Account WHERE Industry = 'Technology'", "help_text": "Salesforce Object Query Language"}
						},
						"max_records": {
							"type": "integer",
							"default": 200,
							"minimum": 1,
							"maximum": 2000,
							"description": "Maximum number of records to return (1-2000, default 200)",
							"x-ui": {"label": "Max records", "placeholder": "200"}
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
							"description": "Task subject line",
							"x-ui": {"label": "Subject", "placeholder": "Follow up call"}
						},
						"what_id": {
							"type": "string",
							"description": "Related record ID (Account, Opportunity, etc.)",
							"x-ui": {"label": "Related to (What)", "help_text": "Record ID of the related object (Account, Opportunity, etc.)"}
						},
						"who_id": {
							"type": "string",
							"description": "Related Contact or Lead ID",
							"x-ui": {"label": "Related to (Who)", "help_text": "Record ID of a Contact or Lead"}
						},
						"status": {
							"type": "string",
							"default": "Not Started",
							"description": "Task status (default: 'Not Started')",
							"x-ui": {"label": "Status", "placeholder": "Not Started"}
						},
						"priority": {
							"type": "string",
							"default": "Normal",
							"description": "Task priority (e.g. High, Normal, Low)",
							"x-ui": {"label": "Priority", "placeholder": "Normal"}
						},
						"due_date": {
							"type": "string",
							"format": "date",
							"description": "Due date in YYYY-MM-DD format",
							"x-ui": {"widget": "date", "label": "Due date", "help_text": "Expected completion date"}
						},
						"description": {
							"type": "string",
							"description": "Task description or notes",
							"x-ui": {"label": "Description", "widget": "textarea"}
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
							"description": "The Salesforce record ID to attach the note to",
							"x-ui": {"label": "Parent record", "help_text": "Record ID to attach the note to"}
						},
						"title": {
							"type": "string",
							"description": "Note title",
							"x-ui": {"label": "Title", "placeholder": "Meeting notes"}
						},
						"body": {
							"type": "string",
							"description": "Note body content (plain text)",
							"x-ui": {"label": "Note body", "widget": "textarea"}
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
							"description": "Opportunity name",
							"x-ui": {"label": "Name", "placeholder": "Acme Corp - Enterprise Deal"}
						},
						"stage_name": {
							"type": "string",
							"description": "Sales stage (e.g. Prospecting, Qualification, Closed Won)",
							"x-ui": {"label": "Stage", "placeholder": "Prospecting", "help_text": "Must match a valid opportunity stage in your org"}
						},
						"close_date": {
							"type": "string",
							"description": "Expected close date in YYYY-MM-DD format",
							"x-ui": {"label": "Close date", "widget": "date"}
						},
						"amount": {
							"type": "number",
							"description": "Opportunity amount (revenue)",
							"x-ui": {"label": "Amount", "placeholder": "50000", "help_text": "Deal value in your org's currency"}
						},
						"account_id": {
							"type": "string",
							"description": "Related Account record ID (15 or 18 characters)",
							"x-ui": {"label": "Account", "help_text": "Account record ID"}
						},
						"description": {
							"type": "string",
							"description": "Opportunity description",
							"x-ui": {"label": "Description", "widget": "textarea"}
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.update_opportunity",
				Name:        "Update Opportunity",
				Description: "Update stage, amount, close date, name, or description on an existing Opportunity",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["record_id"],
					"properties": {
						"record_id": {
							"type": "string",
							"description": "Opportunity record ID (15 or 18 characters)",
							"x-ui": {"label": "Record ID", "placeholder": "001xx000003DGbYAAW", "help_text": "15 or 18-character Salesforce record ID"}
						},
						"stage_name": {
							"type": "string",
							"description": "New sales stage",
							"x-ui": {"label": "Stage", "placeholder": "Prospecting", "help_text": "Must match a valid opportunity stage in your org"}
						},
						"amount": {
							"type": "number",
							"description": "Updated opportunity amount",
							"x-ui": {"label": "Amount", "placeholder": "50000", "help_text": "Deal value in your org's currency"}
						},
						"close_date": {
							"type": "string",
							"description": "Updated close date in YYYY-MM-DD format",
							"x-ui": {"label": "Close date", "widget": "date"}
						},
						"name": {
							"type": "string",
							"description": "Updated opportunity name",
							"x-ui": {"label": "Name", "placeholder": "Acme Corp - Enterprise Deal"}
						},
						"description": {
							"type": "string",
							"description": "Updated description",
							"x-ui": {"label": "Description", "widget": "textarea"}
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
							"description": "Lead last name",
							"x-ui": {"label": "Last name", "placeholder": "Smith"}
						},
						"company": {
							"type": "string",
							"description": "Lead company name",
							"x-ui": {"label": "Company", "placeholder": "Acme Corp"}
						},
						"first_name": {
							"type": "string",
							"description": "Lead first name",
							"x-ui": {"label": "First name", "placeholder": "Jane"}
						},
						"email": {
							"type": "string",
							"description": "Lead email address",
							"x-ui": {"label": "Email", "placeholder": "jane@acme.com"}
						},
						"phone": {
							"type": "string",
							"description": "Lead phone number",
							"x-ui": {"label": "Phone", "placeholder": "+1 (555) 123-4567"}
						},
						"title": {
							"type": "string",
							"description": "Lead job title",
							"x-ui": {"label": "Title", "placeholder": "VP of Sales"}
						},
						"lead_source": {
							"type": "string",
							"description": "Lead source (e.g. Web, Phone Inquiry, Partner)",
							"x-ui": {"label": "Lead source", "placeholder": "Web"}
						},
						"status": {
							"type": "string",
							"description": "Lead status (e.g. Open - Not Contacted, Working, Closed - Converted)",
							"x-ui": {"label": "Status", "placeholder": "Open - Not Contacted"}
						},
						"website": {
							"type": "string",
							"description": "Lead company website",
							"x-ui": {"label": "Website", "placeholder": "https://acme.com"}
						},
						"industry": {
							"type": "string",
							"description": "Lead industry",
							"x-ui": {"label": "Industry", "placeholder": "Technology"}
						}
					}
				}`)),
			},
			{
				ActionType:  "salesforce.convert_lead",
				Name:        "Convert Lead",
				Description: "Convert a Lead to an Account, Contact, and optionally an Opportunity (irreversible)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["lead_id", "converted_status"],
					"properties": {
						"lead_id": {
							"type": "string",
							"description": "Lead record ID to convert (15 or 18 characters)",
							"x-ui": {"label": "Lead ID", "placeholder": "00Qxx000001abcDEFG", "help_text": "15 or 18-character Salesforce record ID"}
						},
						"converted_status": {
							"type": "string",
							"description": "Lead status to set after conversion (e.g. Closed - Converted)",
							"x-ui": {"label": "Converted status", "help_text": "Must be a valid converted lead status in your org"}
						},
						"account_id": {
							"type": "string",
							"description": "Existing Account ID to merge into (omit to create new account)",
							"x-ui": {"label": "Account", "help_text": "Account record ID"}
						},
						"contact_id": {
							"type": "string",
							"description": "Existing Contact ID to merge into (omit to create new contact)",
							"x-ui": {"label": "Contact", "help_text": "Contact record ID"}
						},
						"opportunity_name": {
							"type": "string",
							"description": "Name for the new Opportunity (ignored if do_not_create_opportunity is true)",
							"x-ui": {"label": "Opportunity name", "placeholder": "Acme Corp - Enterprise Deal"}
						},
						"do_not_create_opportunity": {
							"type": "boolean",
							"description": "Set to true to skip creating an Opportunity during conversion",
							"x-ui": {"label": "Skip opportunity creation", "widget": "toggle"}
						},
						"owner_id": {
							"type": "string",
							"description": "User ID to assign as owner of the converted records",
							"x-ui": {"label": "Owner", "help_text": "User record ID"}
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
							"description": "Salesforce object type (e.g. Lead, Contact, Account, Opportunity)",
							"x-ui": {"label": "Object type", "placeholder": "Lead", "help_text": "Salesforce object API name — e.g. Lead, Contact, Account, Opportunity"}
						},
						"record_id": {
							"type": "string",
							"description": "The 15 or 18-character Salesforce record ID to delete",
							"x-ui": {"label": "Record ID", "placeholder": "001xx000003DGbYAAW", "help_text": "15 or 18-character Salesforce record ID"}
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
							"description": "Salesforce object type to describe (e.g. Lead, Opportunity, Account, or any custom object)",
							"x-ui": {"label": "Object type", "placeholder": "Lead", "help_text": "Salesforce object API name — e.g. Lead, Contact, Account, Opportunity"}
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
							"description": "Report record ID (15 or 18 characters)",
							"x-ui": {"label": "Report ID", "help_text": "18-character report ID — find in the report URL"}
						},
						"include_details": {
							"type": "boolean",
							"description": "Include detailed row-level data in the response (default: false, summary only)",
							"x-ui": {"label": "Include details", "widget": "toggle"}
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
				Description: "Agent can create Opportunity records (name, stage, close date, amount, account, description).",
				Parameters:  json.RawMessage(`{"name":"*","stage_name":"*","close_date":"*","amount":"*","account_id":"*","description":"*"}`),
			},
			{
				ID:          "tpl_salesforce_update_opportunity_any",
				ActionType:  "salesforce.update_opportunity",
				Name:        "Update opportunities",
				Description: "Agent can update Opportunity stage, amount, close date, name, or description.",
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
				ID:          "tpl_salesforce_convert_lead_any",
				ActionType:  "salesforce.convert_lead",
				Name:        "Convert leads",
				Description: "Agent can convert Leads to Accounts, Contacts, and Opportunities. This action is irreversible.",
				Parameters:  json.RawMessage(`{"lead_id":"*","converted_status":"*","account_id":"*","contact_id":"*","opportunity_name":"*","do_not_create_opportunity":"*","owner_id":"*"}`),
			},
			{
				ID:          "tpl_salesforce_delete_record_any",
				ActionType:  "salesforce.delete_record",
				Name:        "Delete any record",
				Description: "Agent can permanently delete any Salesforce record by ID. Use with caution.",
				Parameters:  json.RawMessage(`{"sobject_type":"*","record_id":"*"}`),
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
