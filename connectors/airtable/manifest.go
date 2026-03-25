package airtable

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

func (c *AirtableConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "airtable",
		Name:        "Airtable",
		Description: "Airtable integration for structured data and no-code databases",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "airtable.list_bases",
				Name:        "List Bases",
				Description: "List all bases accessible to the authenticated user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"offset": {
							"type": "string",
							"description": "Pagination offset from a previous response",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.list_records",
				Name:        "List Records",
				Description: "List records from an Airtable table with optional filtering, sorting, and pagination",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app'). Find this in your Airtable URL: airtable.com/appXXX/...",
							"x-ui": {"label": "Base ID", "placeholder": "appABC123def456", "help_text": "Starts with 'app' — find in your Airtable base URL"}
						},
						"table": {
							"type": "string",
							"description": "Table name (e.g. 'Tasks') or table ID (starts with 'tbl'). Visible in the tab bar of your Airtable base.",
							"x-ui": {"label": "Table", "placeholder": "Tasks", "help_text": "Table name from the tab bar, or table ID starting with 'tbl'"}
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Only return these fields (column names). Omit to return all fields.",
							"x-ui": {"label": "Fields", "help_text": "Column names to include"}
						},
						"filter_by_formula": {
							"type": "string",
							"description": "Airtable formula to filter records. Examples: \"{Status} = 'Active'\", \"AND({Priority} = 'High', {Assignee} != '')\". See https://support.airtable.com/docs/formula-field-reference",
							"x-ui": {"label": "Filter by formula", "placeholder": "{Status} = 'Active'", "help_text": "Airtable formula syntax — see https://support.airtable.com/docs/formula-field-reference", "widget": "textarea"}
						},
						"max_records": {
							"type": "integer",
							"description": "Maximum total records to return. Omit for Airtable's default (all matching records, paginated).",
							"x-ui": {"label": "Max records"}
						},
						"page_size": {
							"type": "integer",
							"description": "Records per page (1-100, default 100)",
							"x-ui": {"label": "Page size"}
						},
						"sort": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["field"],
								"properties": {
									"field": {"type": "string", "description": "Field name to sort by", "x-ui": {"label": "Field", "placeholder": "Name"}},
									"direction": {"type": "string", "enum": ["asc", "desc"], "description": "Sort direction (default: asc)", "x-ui": {"label": "Direction", "widget": "select"}}
								}
							},
							"description": "Sort order for records",
							"x-ui": {"label": "Sort"}
						},
						"view": {
							"type": "string",
							"description": "Name or ID of a view to filter/sort by. Applies the view's filters and sorts before any additional parameters.",
							"x-ui": {"label": "View", "placeholder": "Grid view"}
						},
						"offset": {
							"type": "string",
							"description": "Pagination offset from a previous response",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.get_record",
				Name:        "Get Record",
				Description: "Get a single record by its ID",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "record_id"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app'). Find this in your Airtable URL: airtable.com/appXXX/...",
							"x-ui": {"label": "Base ID", "placeholder": "appABC123def456", "help_text": "Starts with 'app' — find in your Airtable base URL"}
						},
						"table": {
							"type": "string",
							"description": "Table name (e.g. 'Tasks') or table ID (starts with 'tbl'). Visible in the tab bar of your Airtable base.",
							"x-ui": {"label": "Table", "placeholder": "Tasks", "help_text": "Table name from the tab bar, or table ID starting with 'tbl'"}
						},
						"record_id": {
							"type": "string",
							"description": "Record ID (starts with 'rec'). Visible when expanding a record in Airtable.",
							"x-ui": {"label": "Record ID", "placeholder": "recABC123def456", "help_text": "Starts with 'rec' — visible in the expanded record URL"}
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.create_records",
				Name:        "Create Records",
				Description: "Create one or more records in an Airtable table (batch up to 10)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "records"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app'). Find this in your Airtable URL: airtable.com/appXXX/...",
							"x-ui": {"label": "Base ID", "placeholder": "appABC123def456", "help_text": "Starts with 'app' — find in your Airtable base URL"}
						},
						"table": {
							"type": "string",
							"description": "Table name (e.g. 'Tasks') or table ID (starts with 'tbl'). Visible in the tab bar of your Airtable base.",
							"x-ui": {"label": "Table", "placeholder": "Tasks", "help_text": "Table name from the tab bar, or table ID starting with 'tbl'"}
						},
						"records": {
							"type": "array",
							"minItems": 1,
							"maxItems": 10,
							"items": {
								"type": "object",
								"required": ["fields"],
								"properties": {
									"fields": {
										"type": "object",
										"description": "Field name-value pairs for the record (e.g. {\"Name\": \"John\", \"Email\": \"john@example.com\"})",
										"x-ui": {"label": "Fields"}
									}
								}
							},
							"description": "Records to create (1-10). Airtable limits batch operations to 10 records per request.",
							"x-ui": {"label": "Records", "help_text": "Up to 10 records per batch request"}
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.update_records",
				Name:        "Update Records",
				Description: "Update one or more existing records with partial updates via PATCH (batch up to 10). Only specified fields are modified; unspecified fields are left unchanged.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "records"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app'). Find this in your Airtable URL: airtable.com/appXXX/...",
							"x-ui": {"label": "Base ID", "placeholder": "appABC123def456", "help_text": "Starts with 'app' — find in your Airtable base URL"}
						},
						"table": {
							"type": "string",
							"description": "Table name (e.g. 'Tasks') or table ID (starts with 'tbl'). Visible in the tab bar of your Airtable base.",
							"x-ui": {"label": "Table", "placeholder": "Tasks", "help_text": "Table name from the tab bar, or table ID starting with 'tbl'"}
						},
						"records": {
							"type": "array",
							"minItems": 1,
							"maxItems": 10,
							"items": {
								"type": "object",
								"required": ["id", "fields"],
								"properties": {
									"id": {
										"type": "string",
										"description": "Record ID to update (starts with 'rec')",
										"x-ui": {"label": "Record ID", "placeholder": "recABC123def456", "help_text": "Starts with 'rec' — visible in the expanded record URL"}
									},
									"fields": {
										"type": "object",
										"description": "Field name-value pairs to update. Only specified fields are changed.",
										"x-ui": {"label": "Fields"}
									}
								}
							},
							"description": "Records to update (1-10). Airtable limits batch operations to 10 records per request.",
							"x-ui": {"label": "Records", "help_text": "Up to 10 records per batch request"}
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.delete_records",
				Name:        "Delete Records",
				Description: "Permanently delete one or more records from an Airtable table (batch up to 10). This action cannot be undone.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "record_ids"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app'). Find this in your Airtable URL: airtable.com/appXXX/...",
							"x-ui": {"label": "Base ID", "placeholder": "appABC123def456", "help_text": "Starts with 'app' — find in your Airtable base URL"}
						},
						"table": {
							"type": "string",
							"description": "Table name (e.g. 'Tasks') or table ID (starts with 'tbl'). Visible in the tab bar of your Airtable base.",
							"x-ui": {"label": "Table", "placeholder": "Tasks", "help_text": "Table name from the tab bar, or table ID starting with 'tbl'"}
						},
						"record_ids": {
							"type": "array",
							"minItems": 1,
							"maxItems": 10,
							"items": {"type": "string"},
							"description": "Record IDs to delete (each starts with 'rec'). Airtable limits batch operations to 10 records per request.",
							"x-ui": {"label": "Record IDs", "help_text": "Up to 10 record IDs starting with 'rec'"}
						}
					}
				}`)),
			},
			{
				ActionType:  "airtable.search_records",
				Name:        "Search Records",
				Description: "Search records using an Airtable formula filter. Convenience wrapper around list_records with a required formula and a default limit of 100 records.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["base_id", "table", "formula"],
					"properties": {
						"base_id": {
							"type": "string",
							"description": "Airtable base ID (starts with 'app'). Find this in your Airtable URL: airtable.com/appXXX/...",
							"x-ui": {"label": "Base ID", "placeholder": "appABC123def456", "help_text": "Starts with 'app' — find in your Airtable base URL"}
						},
						"table": {
							"type": "string",
							"description": "Table name (e.g. 'Tasks') or table ID (starts with 'tbl'). Visible in the tab bar of your Airtable base.",
							"x-ui": {"label": "Table", "placeholder": "Tasks", "help_text": "Table name from the tab bar, or table ID starting with 'tbl'"}
						},
						"formula": {
							"type": "string",
							"description": "Airtable formula to filter records. Examples: \"SEARCH('John', {Name})\", \"{Status} = 'Active'\", \"AND({Priority} = 'High', {Due} < TODAY())\". See https://support.airtable.com/docs/formula-field-reference",
							"x-ui": {"label": "Formula", "placeholder": "{Status} = 'Active'", "help_text": "Airtable formula syntax — see https://support.airtable.com/docs/formula-field-reference", "widget": "textarea"}
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Only return these fields (column names). Omit to return all fields.",
							"x-ui": {"label": "Fields", "help_text": "Column names to include"}
						},
						"max_records": {
							"type": "integer",
							"description": "Maximum total records to return (default: 100)",
							"x-ui": {"label": "Max records"}
						},
						"sort": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["field"],
								"properties": {
									"field": {"type": "string", "description": "Field name to sort by", "x-ui": {"label": "Field", "placeholder": "Name"}},
									"direction": {"type": "string", "enum": ["asc", "desc"], "description": "Sort direction (default: asc)", "x-ui": {"label": "Direction", "widget": "select"}}
								}
							},
							"description": "Sort order for results",
							"x-ui": {"label": "Sort"}
						},
						"offset": {
							"type": "string",
							"description": "Pagination offset from a previous response",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "airtable",
				AuthType:      "oauth2",
				OAuthProvider: "airtable",
				OAuthScopes: []string{
					"data.records:read",
					"data.records:write",
					"data.recordComments:read",
					"data.recordComments:write",
					"schema.bases:read",
					"schema.bases:write",
				},
			},
			{Service: "airtable", AuthType: "api_key", InstructionsURL: "https://airtable.com/create/tokens"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_airtable_list_bases",
				ActionType:  "airtable.list_bases",
				Name:        "List all bases",
				Description: "Agent can list all accessible Airtable bases.",
				Parameters:  json.RawMessage(`{"offset":"*"}`),
			},
			{
				ID:          "tpl_airtable_read_records",
				ActionType:  "airtable.list_records",
				Name:        "Read records from any table",
				Description: "Agent can read records from any table in any base.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","fields":"*","filter_by_formula":"*","max_records":"*","page_size":"*","sort":"*","view":"*","offset":"*"}`),
			},
			{
				ID:          "tpl_airtable_get_record",
				ActionType:  "airtable.get_record",
				Name:        "Get any record",
				Description: "Agent can get any record by ID from any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","record_id":"*"}`),
			},
			{
				ID:          "tpl_airtable_create_records",
				ActionType:  "airtable.create_records",
				Name:        "Create records",
				Description: "Agent can create records in any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","records":"*"}`),
			},
			{
				ID:          "tpl_airtable_update_records",
				ActionType:  "airtable.update_records",
				Name:        "Update records",
				Description: "Agent can update records in any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","records":"*"}`),
			},
			{
				ID:          "tpl_airtable_delete_records",
				ActionType:  "airtable.delete_records",
				Name:        "Delete records",
				Description: "Agent can delete records from any table.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","record_ids":"*"}`),
			},
			{
				ID:          "tpl_airtable_search_records",
				ActionType:  "airtable.search_records",
				Name:        "Search records",
				Description: "Agent can search records in any table using formulas.",
				Parameters:  json.RawMessage(`{"base_id":"*","table":"*","formula":"*","fields":"*","max_records":"*","sort":"*"}`),
			},
		},
	}
}
