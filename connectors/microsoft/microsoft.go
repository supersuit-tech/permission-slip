// Package microsoft implements the Microsoft connector for the Permission Slip
// connector execution layer. It uses the Microsoft Graph API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package microsoft

import (
	_ "embed"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://graph.microsoft.com/v1.0"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "access_token"

	// defaultRetryAfter is used when the Graph API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second

	// maxResponseBodySize limits the amount of data read from a Graph API
	// response to prevent unbounded memory consumption from malicious or
	// misbehaving responses. 10 MB is generous for JSON API responses.
	maxResponseBodySize = 10 * 1024 * 1024
)

// MicrosoftConnector owns the shared HTTP client and base URL used by all
// Microsoft actions. Actions hold a pointer back to the connector to access
// these shared resources.
type MicrosoftConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a MicrosoftConnector with sensible defaults (30s timeout,
// https://graph.microsoft.com/v1.0 base URL).
func New() *MicrosoftConnector {
	return &MicrosoftConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a MicrosoftConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *MicrosoftConnector {
	return &MicrosoftConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "microsoft", matching the connectors.id in the database.
func (c *MicrosoftConnector) ID() string { return "microsoft" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *MicrosoftConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "microsoft",
		Name:        "Microsoft",
		Description: "Microsoft 365 integration for email, calendar, OneDrive, Teams, presentations, and Excel via Microsoft Graph API",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "microsoft.send_email",
				Name:        "Send Email",
				Description: "Send an email using Microsoft 365",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "subject", "body"],
					"properties": {
						"to": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Recipient email addresses"
						},
						"subject": {
							"type": "string",
							"description": "Email subject line"
						},
						"body": {
							"type": "string",
							"description": "Email body (HTML or plain text — auto-detected)"
						},
						"cc": {
							"type": "array",
							"items": {"type": "string"},
							"description": "CC recipient email addresses"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_emails",
				Name:        "List Emails",
				Description: "List recent emails from the user's mailbox",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"folder": {
							"type": "string",
							"default": "inbox",
							"description": "Mail folder to list (e.g. inbox, sentitems, drafts)"
						},
						"top": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of emails to return (max 50)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.create_calendar_event",
				Name:        "Create Calendar Event",
				Description: "Create a new event on the user's calendar",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["subject", "start", "end"],
					"properties": {
						"subject": {
							"type": "string",
							"description": "Event subject/title"
						},
						"start": {
							"type": "string",
							"description": "Start date/time in ISO 8601 format (e.g. 2024-01-15T09:00:00)"
						},
						"end": {
							"type": "string",
							"description": "End date/time in ISO 8601 format (e.g. 2024-01-15T10:00:00)"
						},
						"time_zone": {
							"type": "string",
							"default": "UTC",
							"description": "Time zone for start/end times (e.g. America/New_York)"
						},
						"body": {
							"type": "string",
							"description": "Event body/description (HTML supported)"
						},
						"attendees": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Attendee email addresses"
						},
						"location": {
							"type": "string",
							"description": "Event location"
						},
						"calendar_id": {
							"type": "string",
							"description": "Microsoft Graph calendar ID. Leave empty to use the default calendar.",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/calendars",
								"remote_select_id_key": "id",
								"remote_select_label_key": "name",
								"remote_select_fallback_placeholder": "Enter calendar ID (optional)",
								"help_text": "Connect a credential to select a calendar.",
								"label": "Calendar"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_calendar_events",
				Name:        "List Calendar Events",
				Description: "List upcoming events from the user's calendar",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"calendar_id": {
							"type": "string",
							"description": "Microsoft Graph calendar ID. Leave empty to use the default calendar.",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/calendars",
								"remote_select_id_key": "id",
								"remote_select_label_key": "name",
								"remote_select_fallback_placeholder": "Enter calendar ID (optional)",
								"help_text": "Connect a credential to select a calendar.",
								"label": "Calendar"
							}
						},
						"top": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of events to return (max 50)"
						}
					}
				}`)),
			},
			{
			ActionType:  "microsoft.list_drive_files",
				Name:        "List Drive Files",
				Description: "List files and folders in OneDrive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"folder_path": {
							"type": "string",
							"description": "Relative folder path (e.g. Documents/Work). Defaults to root."
						},
						"top": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of items to return (max 50)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.get_drive_file",
				Name:        "Get Drive File",
				Description: "Get file metadata and optionally download text content from OneDrive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID"
						},
						"include_content": {
							"type": "boolean",
							"default": false,
							"description": "Download file content (text files only)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.upload_drive_file",
				Name:        "Upload Drive File",
				Description: "Upload or create a file in OneDrive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_path"],
					"properties": {
						"file_path": {
							"type": "string",
							"description": "Relative file path (e.g. Documents/report.txt)"
						},
						"content": {
							"type": "string",
							"description": "File content to upload (max 4 MB). Omit to create an empty file."
						},
						"conflict_behavior": {
							"type": "string",
							"enum": ["rename", "replace", "fail"],
							"default": "rename",
							"description": "Behavior when a file with the same name exists"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.delete_drive_file",
				Name:        "Delete Drive File",
				Description: "Move a file to the OneDrive recycle bin (recoverable)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.create_document",
				Name:        "Create Document",
				Description: "Create a new Word document in OneDrive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["filename"],
					"properties": {
						"filename": {
							"type": "string",
							"description": "Name for the document (.docx appended if missing)",
							"examples": ["quarterly-report.docx", "meeting-notes"]
						},
						"folder_path": {
							"type": "string",
							"description": "OneDrive folder path (defaults to root)",
							"examples": ["Documents", "Projects/2024"]
						},
						"content": {
							"type": "string",
							"description": "Initial plain-text document content (max 4 MB)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.get_document",
				Name:        "Get Document",
				Description: "Get metadata of a Word document from OneDrive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the document (returned by create or list)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.update_document",
				Name:        "Update Document",
				Description: "Update the content of a Word document in OneDrive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "content"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the document (returned by create or list)"
						},
						"content": {
							"type": "string",
							"description": "New document content (max 4 MB)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_documents",
				Name:        "List Documents",
				Description: "List Word documents from OneDrive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"folder_path": {
							"type": "string",
							"description": "OneDrive folder path (defaults to root)",
							"examples": ["Documents", "Projects/2024"]
						},
						"top": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of documents to return (max 50)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_teams",
				Name:        "List Teams",
				Description: "List Microsoft Teams the user is a member of",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"top": {
							"type": "integer",
							"default": 20,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of teams to return (max 50)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_channels",
				Name:        "List Channels",
				Description: "List channels in a Microsoft Teams team",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id"],
					"properties": {
						"team_id": {
							"type": "string",
							"description": "The ID of the team to list channels for"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.send_channel_message",
				Name:        "Send Channel Message",
				Description: "Send a message to a Microsoft Teams channel",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id", "channel_id", "message"],
					"properties": {
						"team_id": {
							"type": "string",
							"description": "The ID of the team"
						},
						"channel_id": {
							"type": "string",
							"description": "The ID of the channel to post to"
						},
						"message": {
							"type": "string",
							"description": "Message content (HTML or plain text — auto-detected)"
						},
						"reply_to_message_id": {
							"type": "string",
							"description": "Optional message ID to reply to (creates a threaded reply)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_channel_messages",
				Name:        "List Channel Messages",
				Description: "List recent messages from a Microsoft Teams channel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id", "channel_id"],
					"properties": {
						"team_id": {
							"type": "string",
							"description": "The ID of the team"
						},
						"channel_id": {
							"type": "string",
							"description": "The ID of the channel to read messages from"
						},
						"top": {
							"type": "integer",
							"default": 20,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of messages to return (max 50)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.create_presentation",
				Name:        "Create Presentation",
				Description: "Create a new PowerPoint presentation in OneDrive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["filename"],
					"properties": {
						"filename": {
							"type": "string",
							"description": "Name for the presentation file (.pptx extension added if missing)"
						},
						"folder_path": {
							"type": "string",
							"description": "OneDrive folder path (e.g. Documents/Presentations). Defaults to root."
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.list_presentations",
				Name:        "List Presentations",
				Description: "Search for PowerPoint presentations in OneDrive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"folder_path": {
							"type": "string",
							"description": "OneDrive folder path to search in. Defaults to searching all files."
						},
						"top": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 50,
							"description": "Number of presentations to return (max 50)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.get_presentation",
				Name:        "Get Presentation",
				Description: "Get metadata about a specific PowerPoint presentation",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the presentation"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.create_spreadsheet",
				Name:        "Create Spreadsheet",
				Description: "Create a new Excel spreadsheet in OneDrive",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["filename"],
					"properties": {
						"filename": {
							"type": "string",
							"description": "Name for the spreadsheet file (.xlsx extension added if missing)"
						},
						"folder_path": {
							"type": "string",
							"description": "OneDrive folder path (e.g. Documents/Finance). Defaults to root."
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.excel_list_worksheets",
				Name:        "List Excel Worksheets",
				Description: "List all worksheets in an Excel workbook stored in OneDrive",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the Excel workbook"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.excel_read_range",
				Name:        "Read Excel Range",
				Description: "Read cell values from a worksheet range in an Excel workbook",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "sheet_name", "range"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the Excel workbook"
						},
						"sheet_name": {
							"type": "string",
							"description": "Name of the worksheet to read from"
						},
						"range": {
							"type": "string",
							"description": "Cell range to read (e.g. A1:C10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.excel_write_range",
				Name:        "Write Excel Range",
				Description: "Write cell values to a worksheet range in an Excel workbook",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "sheet_name", "range", "values"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the Excel workbook"
						},
						"sheet_name": {
							"type": "string",
							"description": "Name of the worksheet to write to"
						},
						"range": {
							"type": "string",
							"description": "Cell range to write (e.g. A1:C3)"
						},
						"values": {
							"type": "array",
							"items": {
								"type": "array",
								"items": {}
							},
							"description": "2D array of cell values to write"
						}
					}
				}`)),
			},
			{
				ActionType:  "microsoft.excel_append_rows",
				Name:        "Append Excel Rows",
				Description: "Append rows to a named table in an Excel workbook",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "table_name", "values"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "OneDrive item ID of the Excel workbook"
						},
						"table_name": {
							"type": "string",
							"description": "Name of the table to append rows to"
						},
						"values": {
							"type": "array",
							"items": {
								"type": "array",
								"items": {}
							},
							"description": "2D array of row values to append"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "microsoft",
				AuthType:      "oauth2",
				OAuthProvider: "microsoft",
				OAuthScopes: []string{
					"Mail.Send",
					"Mail.Read",
					"Calendars.ReadWrite",
					"Files.ReadWrite",
					"Team.ReadBasic.All",
					"Channel.ReadBasic.All",
					"ChannelMessage.Send",
					"ChannelMessage.Read.All",
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_microsoft_send_email",
				ActionType:  "microsoft.send_email",
				Name:        "Send emails",
				Description: "Agent can send emails to any recipient with any subject and body.",
				Parameters:  json.RawMessage(`{"to":"*","subject":"*","body":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_emails",
				ActionType:  "microsoft.list_emails",
				Name:        "Read inbox",
				Description: "Agent can list emails from the inbox.",
				Parameters:  json.RawMessage(`{"folder":"inbox","top":"*"}`),
			},
			{
				ID:          "tpl_microsoft_create_event",
				ActionType:  "microsoft.create_calendar_event",
				Name:        "Create calendar events",
				Description: "Agent can create events on the calendar with any details.",
				Parameters:  json.RawMessage(`{"subject":"*","start":"*","end":"*","time_zone":"*","body":"*","attendees":"*","location":"*","calendar_id":""}`),
			},
			{
				ID:          "tpl_microsoft_list_events",
				ActionType:  "microsoft.list_calendar_events",
				Name:        "View calendar",
				Description: "Agent can view upcoming calendar events.",
				Parameters:  json.RawMessage(`{"top":"*","calendar_id":""}`),
			},
			{
			ID:          "tpl_microsoft_list_drive_files",
				ActionType:  "microsoft.list_drive_files",
				Name:        "Browse OneDrive files",
				Description: "Agent can list files and folders in OneDrive.",
				Parameters:  json.RawMessage(`{"folder_path":"*","top":"*"}`),
			},
			{
				ID:          "tpl_microsoft_get_drive_file",
				ActionType:  "microsoft.get_drive_file",
				Name:        "Read OneDrive files",
				Description: "Agent can read file metadata and download text content from OneDrive.",
				Parameters:  json.RawMessage(`{"item_id":"*","include_content":"*"}`),
			},
			{
				ID:          "tpl_microsoft_get_drive_file_metadata",
				ActionType:  "microsoft.get_drive_file",
				Name:        "View OneDrive file metadata",
				Description: "Agent can view file metadata from OneDrive but cannot download content.",
				Parameters:  json.RawMessage(`{"item_id":"*","include_content":false}`),
			},
			{
				ID:          "tpl_microsoft_upload_drive_file",
				ActionType:  "microsoft.upload_drive_file",
				Name:        "Upload OneDrive files",
				Description: "Agent can upload files to OneDrive.",
				Parameters:  json.RawMessage(`{"file_path":"*","content":"*","conflict_behavior":"*"}`),
			},
			{
				ID:          "tpl_microsoft_delete_drive_file",
				ActionType:  "microsoft.delete_drive_file",
				Name:        "Delete OneDrive files",
				Description: "Agent can move files to the OneDrive recycle bin.",
				Parameters:  json.RawMessage(`{"item_id":"*"}`),
			},
			{
				ID:          "tpl_microsoft_create_document",
				ActionType:  "microsoft.create_document",
				Name:        "Create Word documents",
				Description: "Agent can create Word documents in OneDrive.",
				Parameters:  json.RawMessage(`{"filename":"*","folder_path":"*","content":"*"}`),
			},
			{
				ID:          "tpl_microsoft_get_document",
				ActionType:  "microsoft.get_document",
				Name:        "Read any document",
				Description: "Agent can read metadata of any document in OneDrive.",
				Parameters:  json.RawMessage(`{"item_id":"*"}`),
			},
			{
				ID:          "tpl_microsoft_update_document",
				ActionType:  "microsoft.update_document",
				Name:        "Edit any document",
				Description: "Agent can update the content of any document in OneDrive.",
				Parameters:  json.RawMessage(`{"item_id":"*","content":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_documents",
				ActionType:  "microsoft.list_documents",
				Name:        "Browse documents",
				Description: "Agent can list Word documents in OneDrive folders.",
				Parameters:  json.RawMessage(`{"folder_path":"*","top":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_teams",
				ActionType:  "microsoft.list_teams",
				Name:        "List teams",
				Description: "Agent can list the Microsoft Teams the user belongs to.",
				Parameters:  json.RawMessage(`{"top":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_channels",
				ActionType:  "microsoft.list_channels",
				Name:        "List channels",
				Description: "Agent can list channels in a specified team.",
				Parameters:  json.RawMessage(`{"team_id":"*"}`),
			},
			{
				ID:          "tpl_microsoft_send_channel_message",
				ActionType:  "microsoft.send_channel_message",
				Name:        "Send channel messages",
				Description: "Agent can send messages to any Teams channel.",
				Parameters:  json.RawMessage(`{"team_id":"*","channel_id":"*","message":"*","reply_to_message_id":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_channel_messages",
				ActionType:  "microsoft.list_channel_messages",
				Name:        "Read channel messages",
				Description: "Agent can read messages from any Teams channel.",
				Parameters:  json.RawMessage(`{"team_id":"*","channel_id":"*","top":"*"}`),
			},
			{
				ID:          "tpl_microsoft_create_presentation",
				ActionType:  "microsoft.create_presentation",
				Name:        "Create presentations",
				Description: "Agent can create new PowerPoint presentations in OneDrive.",
				Parameters:  json.RawMessage(`{"filename":"*","folder_path":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_presentations",
				ActionType:  "microsoft.list_presentations",
				Name:        "List presentations",
				Description: "Agent can search for PowerPoint presentations in OneDrive.",
				Parameters:  json.RawMessage(`{"folder_path":"*","top":"*"}`),
			},
			{
				ID:          "tpl_microsoft_get_presentation",
				ActionType:  "microsoft.get_presentation",
				Name:        "View presentation details",
				Description: "Agent can view metadata about PowerPoint presentations.",
				Parameters:  json.RawMessage(`{"item_id":"*"}`),
			},
			{
				ID:          "tpl_microsoft_excel_list_worksheets",
				ActionType:  "microsoft.excel_list_worksheets",
				Name:        "List Excel worksheets",
				Description: "Agent can list worksheets in a specific workbook.",
				Parameters:  json.RawMessage(`{"item_id":"*"}`),
			},
			{
				ID:          "tpl_microsoft_excel_read_range",
				ActionType:  "microsoft.excel_read_range",
				Name:        "Read Excel range",
				Description: "Agent can read any range from a specific workbook.",
				Parameters:  json.RawMessage(`{"item_id":"*","sheet_name":"*","range":"*"}`),
			},
			{
				ID:          "tpl_microsoft_excel_write_range",
				ActionType:  "microsoft.excel_write_range",
				Name:        "Write Excel range",
				Description: "Agent can write to any range in a specific workbook.",
				Parameters:  json.RawMessage(`{"item_id":"*","sheet_name":"*","range":"*","values":"*"}`),
			},
			{
				ID:          "tpl_microsoft_excel_append_rows",
				ActionType:  "microsoft.excel_append_rows",
				Name:        "Append Excel rows",
				Description: "Agent can append rows to a table in a specific workbook.",
				Parameters:  json.RawMessage(`{"item_id":"*","table_name":"*","values":"*"}`),
			},
			{
				ID:          "tpl_microsoft_excel_read_any",
				ActionType:  "microsoft.excel_read_range",
				Name:        "Read from any workbook",
				Description: "Agent can read from any Excel workbook the user has access to.",
				Parameters:  json.RawMessage(`{"item_id":"*","sheet_name":"*","range":"*"}`),
			},
			{
				ID:          "tpl_microsoft_create_spreadsheet",
				ActionType:  "microsoft.create_spreadsheet",
				Name:        "Create spreadsheets",
				Description: "Agent can create new Excel spreadsheets in OneDrive.",
				Parameters:  json.RawMessage(`{"filename":"*","folder_path":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *MicrosoftConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"microsoft.send_email":            &sendEmailAction{conn: c},
		"microsoft.list_emails":           &listEmailsAction{conn: c},
		"microsoft.create_calendar_event": &createCalendarEventAction{conn: c},
		"microsoft.list_calendar_events":  &listCalendarEventsAction{conn: c},
		"microsoft.list_drive_files":      &listDriveFilesAction{conn: c},
		"microsoft.get_drive_file":        &getDriveFileAction{conn: c},
		"microsoft.upload_drive_file":     &uploadDriveFileAction{conn: c},
		"microsoft.delete_drive_file":     &deleteDriveFileAction{conn: c},
		"microsoft.create_document":       &createDocumentAction{conn: c},
		"microsoft.get_document":          &getDocumentAction{conn: c},
		"microsoft.update_document":       &updateDocumentAction{conn: c},
		"microsoft.list_documents":        &listDocumentsAction{conn: c},
		"microsoft.list_teams":            &listTeamsAction{conn: c},
		"microsoft.list_channels":         &listChannelsAction{conn: c},
		"microsoft.send_channel_message":  &sendChannelMessageAction{conn: c},
		"microsoft.list_channel_messages": &listChannelMessagesAction{conn: c},
		"microsoft.create_presentation":   &createPresentationAction{conn: c},
		"microsoft.list_presentations":    &listPresentationsAction{conn: c},
		"microsoft.get_presentation":      &getPresentationAction{conn: c},
		"microsoft.create_spreadsheet":    &createSpreadsheetAction{conn: c},
		"microsoft.excel_list_worksheets": &excelListWorksheetsAction{conn: c},
		"microsoft.excel_read_range":      &excelReadRangeAction{conn: c},
		"microsoft.excel_write_range":     &excelWriteRangeAction{conn: c},
		"microsoft.excel_append_rows":     &excelAppendRowsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token (provided by the OAuth flow).
func (c *MicrosoftConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// executeRequest is the core request lifecycle shared by all Microsoft Graph
// request methods. It handles:
//   - Sending the HTTP request
//   - Timeout and context cancellation detection
//   - Rate limit detection (HTTP 429 → RateLimitError with Retry-After)
//   - Response body size limiting (maxResponseBodySize) to prevent OOM
//   - Error response mapping via mapGraphError (see response.go)
//
// Returns the raw response body on success. Callers handle request construction
// and response interpretation.
func (c *MicrosoftConnector) executeRequest(req *http.Request) ([]byte, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("Microsoft Graph API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.TimeoutError{Message: "Microsoft Graph API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("Microsoft Graph API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return nil, &connectors.RateLimitError{
			Message:    "Microsoft Graph API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, mapGraphError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// extractToken retrieves the OAuth access token from credentials.
func extractToken(creds connectors.Credentials) (string, error) {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return "", &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}
	return token, nil
}

// doRequest is the shared request lifecycle for all Microsoft Graph JSON actions.
// It handles JSON marshaling/unmarshaling, authorization, and delegates to
// executeRequest for the HTTP lifecycle.
func (c *MicrosoftConnector) doRequest(ctx context.Context, method, path string, creds connectors.Credentials, body any, dest any) error {
	token, err := extractToken(creds)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		payload, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("marshaling request body: %w", marshalErr)
		}
		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	respBody, err := c.executeRequest(req)
	if err != nil {
		return err
	}

	// Some endpoints return 204 No Content (e.g. sendMail).
	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				Message: "failed to decode Microsoft Graph API response",
			}
		}
	}

	return nil
}

// doUpload sends a raw-body request to the Microsoft Graph API.
// Used by OneDrive file upload endpoints (PUT .../content) where the body is
// file content rather than JSON. Delegates to executeRequest for the HTTP lifecycle
// and JSON-unmarshals the response.
func (c *MicrosoftConnector) doUpload(ctx context.Context, method, path string, creds connectors.Credentials, body []byte, contentType string, dest any) error {
	token, err := extractToken(creds)
	if err != nil {
		return err
	}

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	respBody, err := c.executeRequest(req)
	if err != nil {
		return err
	}

	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				Message: "failed to decode Microsoft Graph API response",
			}
		}
	}

	return nil
}

// doRequestRaw is like doRequest but returns the response body as a string
// instead of JSON-unmarshaling it. Used for downloading file content.
func (c *MicrosoftConnector) doRequestRaw(ctx context.Context, method, path string, creds connectors.Credentials) (string, error) {
	token, err := extractToken(creds)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	respBody, execErr := c.executeRequest(req)
	if execErr != nil {
		return "", execErr
	}

	return string(respBody), nil
}

// doPutRaw sends a PUT request with a raw byte body (not JSON-encoded) and returns
// the response body bytes. Used for uploading file content to OneDrive.
func (c *MicrosoftConnector) doPutRaw(ctx context.Context, path string, creds connectors.Credentials, content []byte) ([]byte, error) {
	token, err := extractToken(creds)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	return c.executeRequest(req)
}
