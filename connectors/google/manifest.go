package google

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *GoogleConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "google",
		Name:        "Google",
		Description: "Google integration for Gmail, Calendar, and Sheets",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "google.send_email",
				Name:        "Send Email",
				Description: "Send an email via Gmail",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "subject", "body"],
					"properties": {
						"to": {
							"type": "string",
							"description": "Recipient email address"
						},
						"subject": {
							"type": "string",
							"description": "Email subject line"
						},
						"body": {
							"type": "string",
							"description": "Email body (plain text)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.list_emails",
				Name:        "List Emails",
				Description: "List recent emails from Gmail inbox",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "Gmail search query (e.g. 'from:user@example.com is:unread')"
						},
						"max_results": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of emails to return (1-100, default 10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.create_calendar_event",
				Name:        "Create Calendar Event",
				Description: "Create a new event on Google Calendar",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["summary", "start_time", "end_time"],
					"properties": {
						"summary": {
							"type": "string",
							"description": "Event title"
						},
						"description": {
							"type": "string",
							"description": "Event description"
						},
						"start_time": {
							"type": "string",
							"description": "Start time in RFC 3339 format (e.g. '2024-01-15T09:00:00-05:00')"
						},
						"end_time": {
							"type": "string",
							"description": "End time in RFC 3339 format (e.g. '2024-01-15T10:00:00-05:00')"
						},
						"attendees": {
							"type": "array",
							"items": {"type": "string"},
							"description": "List of attendee email addresses"
						},
						"calendar_id": {
							"type": "string",
							"default": "primary",
							"description": "Calendar ID (defaults to 'primary')"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.list_calendar_events",
				Name:        "List Calendar Events",
				Description: "List upcoming events from Google Calendar",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"calendar_id": {
							"type": "string",
							"default": "primary",
							"description": "Calendar ID (defaults to 'primary')"
						},
						"max_results": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 250,
							"description": "Maximum number of events to return (1-250, default 10)"
						},
						"time_min": {
							"type": "string",
							"description": "Lower bound (inclusive) for event start time in RFC 3339 format. Defaults to now."
						},
						"time_max": {
							"type": "string",
							"description": "Upper bound (exclusive) for event start time in RFC 3339 format"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.sheets_read_range",
				Name:        "Read Spreadsheet Range",
				Description: "Read cell values from a specified range in a Google Sheets spreadsheet",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["spreadsheet_id", "range"],
					"properties": {
						"spreadsheet_id": {
							"type": "string",
							"description": "The ID of the spreadsheet (the long string in the URL between /d/ and /edit)"
						},
						"range": {
							"type": "string",
							"description": "A1 notation range including sheet name (e.g. 'Sheet1!A1:D10'). Use sheet name alone to read all data (e.g. 'Sheet1')."
						}
					}
				}`)),
			},
			{
				ActionType:  "google.sheets_write_range",
				Name:        "Write Spreadsheet Range",
				Description: "Write cell values to a specified range in a Google Sheets spreadsheet",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["spreadsheet_id", "range", "values"],
					"properties": {
						"spreadsheet_id": {
							"type": "string",
							"description": "The ID of the spreadsheet (the long string in the URL between /d/ and /edit)"
						},
						"range": {
							"type": "string",
							"description": "A1 notation range including sheet name (e.g. 'Sheet1!A1:C3'). Defines the top-left starting cell for the write."
						},
						"values": {
							"type": "array",
							"items": {
								"type": "array",
								"items": {}
							},
							"description": "2D array of cell values (rows of columns). All rows must have the same number of columns. Values are parsed as if typed into the UI (formulas and formats applied)."
						}
					}
				}`)),
			},
			{
				ActionType:  "google.sheets_append_rows",
				Name:        "Append Spreadsheet Rows",
				Description: "Append rows to a sheet or table in a Google Sheets spreadsheet",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["spreadsheet_id", "range", "values"],
					"properties": {
						"spreadsheet_id": {
							"type": "string",
							"description": "The ID of the spreadsheet (the long string in the URL between /d/ and /edit)"
						},
						"range": {
							"type": "string",
							"description": "Sheet name or starting cell (e.g. 'Sheet1' or 'Sheet1!A1'). Rows are appended after the last row with data in this range."
						},
						"values": {
							"type": "array",
							"items": {
								"type": "array",
								"items": {}
							},
							"description": "2D array of row values to append (rows of columns). All rows must have the same number of columns. Values are parsed as if typed into the UI."
						}
					}
				}`)),
			},
			{
				ActionType:  "google.sheets_list_sheets",
				Name:        "List Worksheets",
				Description: "List all worksheets (tabs) in a Google Sheets spreadsheet",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["spreadsheet_id"],
					"properties": {
						"spreadsheet_id": {
							"type": "string",
							"description": "The ID of the spreadsheet (the long string in the URL between /d/ and /edit)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "google",
				AuthType:      "oauth2",
				OAuthProvider: "google",
				OAuthScopes: []string{
					"https://www.googleapis.com/auth/gmail.send",
					"https://www.googleapis.com/auth/gmail.readonly",
					"https://www.googleapis.com/auth/calendar.events",
					"https://www.googleapis.com/auth/spreadsheets",
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_google_send_email",
				ActionType:  "google.send_email",
				Name:        "Send emails freely",
				Description: "Agent can send emails to any recipient with any subject and body.",
				Parameters:  json.RawMessage(`{"to":"*","subject":"*","body":"*"}`),
			},
			{
				ID:          "tpl_google_send_email_to_recipient",
				ActionType:  "google.send_email",
				Name:        "Send email to specific recipient",
				Description: "Locks the recipient; agent chooses the subject and body.",
				Parameters:  json.RawMessage(`{"to":"recipient@example.com","subject":"*","body":"*"}`),
			},
			{
				ID:          "tpl_google_list_emails",
				ActionType:  "google.list_emails",
				Name:        "Search emails",
				Description: "Agent can search and list emails from the inbox.",
				Parameters:  json.RawMessage(`{"query":"*","max_results":"*"}`),
			},
			{
				ID:          "tpl_google_list_unread_emails",
				ActionType:  "google.list_emails",
				Name:        "List unread emails",
				Description: "Agent can list unread emails only. Query is locked to is:unread.",
				Parameters:  json.RawMessage(`{"query":"is:unread","max_results":"*"}`),
			},
			{
				ID:          "tpl_google_create_calendar_event",
				ActionType:  "google.create_calendar_event",
				Name:        "Create calendar events",
				Description: "Agent can create events on any calendar.",
				Parameters:  json.RawMessage(`{"summary":"*","description":"*","start_time":"*","end_time":"*","attendees":"*","calendar_id":"*"}`),
			},
			{
				ID:          "tpl_google_create_calendar_event_no_attendees",
				ActionType:  "google.create_calendar_event",
				Name:        "Create personal calendar events",
				Description: "Agent can create events on the primary calendar without inviting attendees.",
				Parameters:  json.RawMessage(`{"summary":"*","description":"*","start_time":"*","end_time":"*","calendar_id":"primary"}`),
			},
			{
				ID:          "tpl_google_list_calendar_events",
				ActionType:  "google.list_calendar_events",
				Name:        "List calendar events",
				Description: "Agent can list upcoming events from any calendar.",
				Parameters:  json.RawMessage(`{"calendar_id":"*","max_results":"*","time_min":"*","time_max":"*"}`),
			},
			{
				ID:          "tpl_google_sheets_read_range",
				ActionType:  "google.sheets_read_range",
				Name:        "Read from specific spreadsheet",
				Description: "Locks the spreadsheet; agent chooses the range to read.",
				Parameters:  json.RawMessage(`{"spreadsheet_id":"SPREADSHEET_ID","range":"*"}`),
			},
			{
				ID:          "tpl_google_sheets_write_range",
				ActionType:  "google.sheets_write_range",
				Name:        "Write to specific spreadsheet",
				Description: "Locks the spreadsheet; agent chooses the range and values to write.",
				Parameters:  json.RawMessage(`{"spreadsheet_id":"SPREADSHEET_ID","range":"*","values":"*"}`),
			},
			{
				ID:          "tpl_google_sheets_append_rows",
				ActionType:  "google.sheets_append_rows",
				Name:        "Append to specific spreadsheet",
				Description: "Locks the spreadsheet; agent chooses the range and rows to append.",
				Parameters:  json.RawMessage(`{"spreadsheet_id":"SPREADSHEET_ID","range":"*","values":"*"}`),
			},
			{
				ID:          "tpl_google_sheets_read_any",
				ActionType:  "google.sheets_read_range",
				Name:        "Read from any spreadsheet",
				Description: "Agent can read from any spreadsheet and any range.",
				Parameters:  json.RawMessage(`{"spreadsheet_id":"*","range":"*"}`),
			},
			{
				ID:          "tpl_google_sheets_list",
				ActionType:  "google.sheets_list_sheets",
				Name:        "List worksheets in any spreadsheet",
				Description: "Agent can list worksheets in any spreadsheet.",
				Parameters:  json.RawMessage(`{"spreadsheet_id":"*"}`),
			},
		},
	}
}
