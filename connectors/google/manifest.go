package google

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *GoogleConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "google",
		Name:        "Google",
		Description: "Google integration for Gmail, Calendar, Slides, Sheets, Docs, Chat, Meet, and Drive",
		LogoSVG:     logoSVG,
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
			// Sheets actions
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
			// Docs actions
			{
				ActionType:  "google.create_document",
				Name:        "Create Document",
				Description: "Create a new Google Doc with a title and optional body content",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Title of the new Google Doc"
						},
						"body": {
							"type": "string",
							"description": "Optional initial body content (plain text)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.get_document",
				Name:        "Get Document",
				Description: "Retrieve the content and metadata of a Google Doc by document ID. Returns plain text content, word count, and a direct link to the document.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["document_id"],
					"properties": {
						"document_id": {
							"type": "string",
							"description": "The ID of the Google Doc to retrieve (e.g. '1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms')"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.update_document",
				Name:        "Update Document",
				Description: "Append or insert text into an existing Google Doc. By default text is appended to the end; specify an index to insert at a specific position.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["document_id", "text"],
					"properties": {
						"document_id": {
							"type": "string",
							"description": "The ID of the Google Doc to update (e.g. '1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgVE2upms')"
						},
						"text": {
							"type": "string",
							"description": "Text to insert into the document"
						},
						"index": {
							"type": "integer",
							"minimum": 1,
							"description": "Character index to insert at (1-based). Defaults to end of document."
						}
					}
				}`)),
			},
			{
				ActionType:  "google.list_documents",
				Name:        "List Documents",
				Description: "Search and list Google Docs from Drive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query to filter documents by name"
						},
						"max_results": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of documents to return (1-100, default 10)"
						}
					}
				}`)),
			},
			// Chat actions
			{
				ActionType:  "google.create_presentation",
				Name:        "Create Presentation",
				Description: "Create a new Google Slides presentation",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Title of the new presentation"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.get_presentation",
				Name:        "Get Presentation",
				Description: "Retrieve metadata about a Google Slides presentation",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["presentation_id"],
					"properties": {
						"presentation_id": {
							"type": "string",
							"description": "The ID of the presentation to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.add_slide",
				Name:        "Add Slide",
				Description: "Add a new slide to an existing Google Slides presentation",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["presentation_id"],
					"properties": {
						"presentation_id": {
							"type": "string",
							"description": "The ID of the presentation to add a slide to"
						},
						"layout": {
							"type": "string",
							"enum": ["BLANK", "TITLE", "TITLE_AND_BODY", "TITLE_ONLY", "SECTION_HEADER", "SECTION_TITLE_AND_DESCRIPTION", "ONE_COLUMN_TEXT", "MAIN_POINT", "BIG_NUMBER", "CAPTION_ONLY", "TITLE_AND_TWO_COLUMNS"],
							"default": "BLANK",
							"description": "Predefined slide layout (defaults to BLANK)"
						},
						"insertion_index": {
							"type": "integer",
							"minimum": 0,
							"description": "Position to insert the slide at (defaults to end)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.send_chat_message",
				Name:        "Send Chat Message",
				Description: "Send a message to a Google Chat space",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["space_name", "text"],
					"properties": {
						"space_name": {
							"type": "string",
							"description": "The resource name of the space (e.g. 'spaces/AAAA1234')"
						},
						"text": {
							"type": "string",
							"description": "The message text"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.list_chat_spaces",
				Name:        "List Chat Spaces",
				Description: "List Google Chat spaces accessible to the user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"page_size": {
							"type": "integer",
							"default": 20,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of spaces to return (1-100, default 20)"
						},
						"filter": {
							"type": "string",
							"description": "Optional filter query (e.g. 'spaceType = \"SPACE\"' to list only named spaces)"
						}
					}
				}`)),
			},
			// Meet action
			{
				ActionType:  "google.create_meeting",
				Name:        "Create Meeting",
				Description: "Create a Google Calendar event with an auto-generated Google Meet link",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["summary", "start_time", "end_time"],
					"properties": {
						"summary": {
							"type": "string",
							"description": "Meeting title"
						},
						"description": {
							"type": "string",
							"description": "Meeting description"
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
			// Drive actions
			{
				ActionType:  "google.list_drive_files",
				Name:        "List Drive Files",
				Description: "List or search files in Google Drive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "Google Drive search query (e.g. \"name contains 'report'\")"
						},
						"max_results": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of files to return (1-100, default 10)"
						},
						"folder_id": {
							"type": "string",
							"description": "Folder ID to list files from (defaults to all accessible files)"
						},
						"order_by": {
							"type": "string",
							"description": "Sort order (e.g. 'modifiedTime desc', 'name'). Defaults to Drive's relevance ordering."
						}
					}
				}`)),
			},
			{
				ActionType:  "google.get_drive_file",
				Name:        "Get Drive File",
				Description: "Get file metadata and optionally download content from Google Drive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_id"],
					"properties": {
						"file_id": {
							"type": "string",
							"description": "The ID of the file to retrieve"
						},
						"include_content": {
							"type": "boolean",
							"default": false,
							"description": "Whether to include file content (exports Google Docs/Sheets/Slides as plain text)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.upload_drive_file",
				Name:        "Upload Drive File",
				Description: "Create and upload a text file to Google Drive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "content"],
					"properties": {
						"name": {
							"type": "string",
							"description": "File name"
						},
						"content": {
							"type": "string",
							"description": "File content (text)"
						},
						"mime_type": {
							"type": "string",
							"default": "text/plain",
							"description": "MIME type of the file (default: text/plain)"
						},
						"folder_id": {
							"type": "string",
							"description": "Parent folder ID (optional)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.delete_drive_file",
				Name:        "Delete Drive File",
				Description: "Move a file to trash in Google Drive (soft delete)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_id"],
					"properties": {
						"file_id": {
							"type": "string",
							"description": "The ID of the file to move to trash"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.update_calendar_event",
				Name:        "Update Calendar Event",
				Description: "Update an existing Google Calendar event (time, title, attendees, or location)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_id"],
					"properties": {
						"event_id": {
							"type": "string",
							"description": "The ID of the calendar event to update"
						},
						"calendar_id": {
							"type": "string",
							"default": "primary",
							"description": "Calendar ID (defaults to 'primary')"
						},
						"summary": {
							"type": "string",
							"description": "New event title"
						},
						"description": {
							"type": "string",
							"description": "New event description"
						},
						"start_time": {
							"type": "string",
							"description": "New start time in RFC 3339 format. Must be provided together with end_time."
						},
						"end_time": {
							"type": "string",
							"description": "New end time in RFC 3339 format. Must be provided together with start_time."
						},
						"attendees": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Replacement list of attendee email addresses. Cannot be used together with clear_attendees."
						},
						"clear_attendees": {
							"type": "boolean",
							"description": "Set to true to remove all attendees from the event. Cannot be used together with attendees."
						},
						"location": {
							"type": "string",
							"description": "Event location"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.delete_calendar_event",
				Name:        "Delete Calendar Event",
				Description: "Delete or cancel a Google Calendar event",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_id"],
					"properties": {
						"event_id": {
							"type": "string",
							"description": "The ID of the calendar event to delete"
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
				ActionType:  "google.search_drive",
				Name:        "Search Drive",
				Description: "Search Google Drive files by name, type, or content",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "Search term to match against file names and content"
						},
						"file_type": {
							"type": "string",
							"enum": ["document", "spreadsheet", "presentation", "folder", "pdf", "image", "video", "audio"],
							"description": "Filter by file type"
						},
						"folder_id": {
							"type": "string",
							"description": "Limit search to files within this folder ID"
						},
						"max_results": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of results to return (1-100, default 10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.create_drive_folder",
				Name:        "Create Drive Folder",
				Description: "Create a new folder in Google Drive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Name of the new folder"
						},
						"parent_id": {
							"type": "string",
							"description": "Parent folder ID (optional, defaults to Drive root)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.send_email_reply",
				Name:        "Reply to Email",
				Description: "Reply to an existing Gmail thread",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["thread_id", "message_id", "body"],
					"properties": {
						"thread_id": {
							"type": "string",
							"description": "The Gmail thread ID to reply to"
						},
						"message_id": {
							"type": "string",
							"description": "The ID of the specific message to reply to (used to fetch subject and recipient)"
						},
						"body": {
							"type": "string",
							"description": "Reply body (plain text)"
						}
					}
				}`)),
			},
			{
				ActionType:  "google.archive_email",
				Name:        "Archive Email",
				Description: "Archive a Gmail message by removing it from the inbox",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["message_id"],
					"properties": {
						"message_id": {
							"type": "string",
							"description": "The Gmail message ID to archive (obtained from list_emails)"
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
					"https://www.googleapis.com/auth/calendar.events",
					"https://www.googleapis.com/auth/presentations",
					"https://www.googleapis.com/auth/spreadsheets",
					"https://www.googleapis.com/auth/documents",
					"https://www.googleapis.com/auth/chat.spaces.readonly",
					"https://www.googleapis.com/auth/chat.messages.create",
					"https://www.googleapis.com/auth/drive",
					"https://www.googleapis.com/auth/gmail.modify",
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
			// Sheets templates
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
			// Docs templates
			{
				ID:          "tpl_google_create_document",
				ActionType:  "google.create_document",
				Name:        "Create documents",
				Description: "Agent can create new Google Docs with any title and body.",
				Parameters:  json.RawMessage(`{"title":"*","body":"*"}`),
			},
			{
				ID:          "tpl_google_create_document_title_only",
				ActionType:  "google.create_document",
				Name:        "Create empty documents",
				Description: "Agent can create new Google Docs with any title but no initial body content.",
				Parameters:  json.RawMessage(`{"title":"*"}`),
			},
			{
				ID:          "tpl_google_get_document",
				ActionType:  "google.get_document",
				Name:        "Read any document",
				Description: "Agent can read the content of any Google Doc by ID.",
				Parameters:  json.RawMessage(`{"document_id":"*"}`),
			},
			{
				ID:          "tpl_google_update_document",
				ActionType:  "google.update_document",
				Name:        "Edit any document",
				Description: "Agent can insert or append text to any Google Doc.",
				Parameters:  json.RawMessage(`{"document_id":"*","text":"*","index":"*"}`),
			},
			{
				ID:          "tpl_google_list_documents",
				ActionType:  "google.list_documents",
				Name:        "Search documents",
				Description: "Agent can search and list Google Docs from Drive.",
				Parameters:  json.RawMessage(`{"query":"*","max_results":"*"}`),
			},
			// Chat templates
			{
				ID:          "tpl_google_create_presentation",
				ActionType:  "google.create_presentation",
				Name:        "Create presentations",
				Description: "Agent can create new Google Slides presentations with any title.",
				Parameters:  json.RawMessage(`{"title":"*"}`),
			},
			{
				ID:          "tpl_google_get_presentation",
				ActionType:  "google.get_presentation",
				Name:        "View presentations",
				Description: "Agent can retrieve metadata about any Google Slides presentation.",
				Parameters:  json.RawMessage(`{"presentation_id":"*"}`),
			},
			{
				ID:          "tpl_google_add_slide",
				ActionType:  "google.add_slide",
				Name:        "Add slides to presentations",
				Description: "Agent can add new slides to any Google Slides presentation.",
				Parameters:  json.RawMessage(`{"presentation_id":"*","layout":"*","insertion_index":"*"}`),
			},
			{
				ID:          "tpl_google_send_chat_message",
				ActionType:  "google.send_chat_message",
				Name:        "Send chat messages",
				Description: "Agent can send messages to any Google Chat space.",
				Parameters:  json.RawMessage(`{"space_name":"*","text":"*"}`),
			},
			{
				ID:          "tpl_google_send_chat_message_to_space",
				ActionType:  "google.send_chat_message",
				Name:        "Send message to specific space",
				Description: "Locks the space; agent chooses the message text.",
				Parameters:  json.RawMessage(`{"space_name":"spaces/EXAMPLE","text":"*"}`),
			},
			{
				ID:          "tpl_google_list_chat_spaces",
				ActionType:  "google.list_chat_spaces",
				Name:        "List chat spaces",
				Description: "Agent can list Google Chat spaces accessible to the user.",
				Parameters:  json.RawMessage(`{"page_size":"*","filter":"*"}`),
			},
			// Meet templates
			{
				ID:          "tpl_google_create_meeting",
				ActionType:  "google.create_meeting",
				Name:        "Create meetings with Meet link",
				Description: "Agent can create calendar events with Google Meet links.",
				Parameters:  json.RawMessage(`{"summary":"*","description":"*","start_time":"*","end_time":"*","attendees":"*","calendar_id":"*"}`),
			},
			{
				ID:          "tpl_google_create_meeting_no_attendees",
				ActionType:  "google.create_meeting",
				Name:        "Create personal meetings",
				Description: "Agent can create meetings on the primary calendar without inviting attendees.",
				Parameters:  json.RawMessage(`{"summary":"*","description":"*","start_time":"*","end_time":"*","calendar_id":"primary"}`),
			},
			// Drive templates
			{
				ID:          "tpl_google_list_drive_files",
				ActionType:  "google.list_drive_files",
				Name:        "Browse Drive files",
				Description: "Agent can list and search files in Google Drive.",
				Parameters:  json.RawMessage(`{"query":"*","max_results":"*","folder_id":"*","order_by":"*"}`),
			},
			{
				ID:          "tpl_google_get_drive_file",
				ActionType:  "google.get_drive_file",
				Name:        "Read Drive files",
				Description: "Agent can read file metadata and content from Google Drive.",
				Parameters:  json.RawMessage(`{"file_id":"*","include_content":"*"}`),
			},
			{
				ID:          "tpl_google_get_drive_file_metadata",
				ActionType:  "google.get_drive_file",
				Name:        "View Drive file metadata",
				Description: "Agent can view file metadata only (no content download).",
				Parameters:  json.RawMessage(`{"file_id":"*","include_content":"false"}`),
			},
			{
				ID:          "tpl_google_upload_drive_file",
				ActionType:  "google.upload_drive_file",
				Name:        "Upload files to Drive",
				Description: "Agent can upload text files to Google Drive.",
				Parameters:  json.RawMessage(`{"name":"*","content":"*","mime_type":"*","folder_id":"*"}`),
			},
			{
				ID:          "tpl_google_upload_drive_file_to_folder",
				ActionType:  "google.upload_drive_file",
				Name:        "Upload files to specific folder",
				Description: "Agent can upload text files to a specific locked folder in Google Drive.",
				Parameters:  json.RawMessage(`{"name":"*","content":"*","mime_type":"*","folder_id":"folder-id-here"}`),
			},
			{
				ID:          "tpl_google_delete_drive_file",
				ActionType:  "google.delete_drive_file",
				Name:        "Trash Drive files",
				Description: "Agent can move files to trash in Google Drive.",
				Parameters:  json.RawMessage(`{"file_id":"*"}`),
			},
			{
				ID:          "tpl_google_update_calendar_event",
				ActionType:  "google.update_calendar_event",
				Name:        "Update calendar events",
				Description: "Agent can update the summary, description, time, attendees, and location of calendar events.",
				Parameters:  json.RawMessage(`{"event_id":"*","calendar_id":"*","summary":"*","description":"*","start_time":"*","end_time":"*","attendees":"*","clear_attendees":"*","location":"*"}`),
			},
			{
				ID:          "tpl_google_update_calendar_event_time",
				ActionType:  "google.update_calendar_event",
				Name:        "Reschedule calendar events",
				Description: "Agent can reschedule events (change start/end time only).",
				Parameters:  json.RawMessage(`{"event_id":"*","calendar_id":"*","start_time":"*","end_time":"*"}`),
			},
			{
				ID:          "tpl_google_delete_calendar_event",
				ActionType:  "google.delete_calendar_event",
				Name:        "Delete calendar events",
				Description: "Agent can delete events from any calendar.",
				Parameters:  json.RawMessage(`{"event_id":"*","calendar_id":"*"}`),
			},
			{
				ID:          "tpl_google_search_drive",
				ActionType:  "google.search_drive",
				Name:        "Search Drive files",
				Description: "Agent can search Drive by name, content, or file type.",
				Parameters:  json.RawMessage(`{"query":"*","file_type":"*","folder_id":"*","max_results":"*"}`),
			},
			{
				ID:          "tpl_google_search_drive_in_folder",
				ActionType:  "google.search_drive",
				Name:        "Search Drive within folder",
				Description: "Agent can search within a specific locked folder.",
				Parameters:  json.RawMessage(`{"query":"*","file_type":"*","folder_id":"folder-id-here","max_results":"*"}`),
			},
			{
				ID:          "tpl_google_create_drive_folder",
				ActionType:  "google.create_drive_folder",
				Name:        "Create Drive folders",
				Description: "Agent can create folders anywhere in Google Drive.",
				Parameters:  json.RawMessage(`{"name":"*","parent_id":"*"}`),
			},
			{
				ID:          "tpl_google_send_email_reply",
				ActionType:  "google.send_email_reply",
				Name:        "Reply to emails",
				Description: "Agent can reply to any existing Gmail thread.",
				Parameters:  json.RawMessage(`{"thread_id":"*","message_id":"*","body":"*"}`),
			},
			{
				ID:          "tpl_google_archive_email",
				ActionType:  "google.archive_email",
				Name:        "Archive emails",
				Description: "Agent can archive any Gmail message by removing it from the inbox.",
				Parameters:  json.RawMessage(`{"message_id":"*"}`),
			},
		},
	}
}
