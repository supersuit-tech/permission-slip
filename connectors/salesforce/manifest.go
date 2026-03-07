package salesforce

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *SalesforceConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "salesforce",
		Name:        "Salesforce",
		Description: "Salesforce CRM integration for managing records, running queries, and logging activities",
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
		},
	}
}
