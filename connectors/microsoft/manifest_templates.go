package microsoft

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// microsoftTemplates returns configuration templates for common Microsoft 365
// use cases. Templates cover Outlook, Calendar, OneDrive, Word, Excel,
// PowerPoint, and Teams — from read-only inbox access to scoped write
// operations.
func microsoftTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		// --- Outlook read ---
		{
			ID:          "tpl_microsoft_list_emails",
			ActionType:  "microsoft.list_emails",
			Name:        "Read inbox",
			Description: "Agent can list emails from the inbox.",
			Parameters:  json.RawMessage(`{"folder":"inbox","top":"*"}`),
		},
		// --- Outlook write ---
		{
			ID:          "tpl_microsoft_send_email",
			ActionType:  "microsoft.send_email",
			Name:        "Send emails",
			Description: "Agent can send emails to any recipient with any subject and body.",
			Parameters:  json.RawMessage(`{"to":"*","subject":"*","body":"*"}`),
		},
		{
			ID:          "tpl_microsoft_send_email_to_domain",
			ActionType:  "microsoft.send_email",
			Name:        "Send email to a domain",
			Description: "Agent can only send emails to recipients matching a specific domain pattern. Set the domain before applying.",
			Parameters:  json.RawMessage(`{"to":{"$pattern":"*@example.com"},"subject":"*","body":"*"}`),
		},
		// --- Calendar read ---
		{
			ID:          "tpl_microsoft_list_events",
			ActionType:  "microsoft.list_calendar_events",
			Name:        "View calendar",
			Description: "Agent can view upcoming calendar events.",
			Parameters:  json.RawMessage(`{"top":"*","calendar_id":""}`),
		},
		// --- Calendar write ---
		{
			ID:          "tpl_microsoft_create_event",
			ActionType:  "microsoft.create_calendar_event",
			Name:        "Create calendar events",
			Description: "Agent can create events on the calendar with any details.",
			Parameters:  json.RawMessage(`{"subject":"*","start":"*","end":"*","time_zone":"*","body":"*","attendees":"*","location":"*","calendar_id":""}`),
		},
		// --- OneDrive read ---
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
			Parameters:  json.RawMessage(`{"item_id":"*","include_content":"false"}`),
		},
		// --- OneDrive write ---
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
		// --- Documents ---
		{
			ID:          "tpl_microsoft_list_documents",
			ActionType:  "microsoft.list_documents",
			Name:        "Browse documents",
			Description: "Agent can list Word documents in OneDrive folders.",
			Parameters:  json.RawMessage(`{"folder_path":"*","top":"*"}`),
		},
		{
			ID:          "tpl_microsoft_get_document",
			ActionType:  "microsoft.get_document",
			Name:        "Read any document",
			Description: "Agent can read metadata of any document in OneDrive.",
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
			ID:          "tpl_microsoft_update_document",
			ActionType:  "microsoft.update_document",
			Name:        "Edit any document",
			Description: "Agent can update the content of any document in OneDrive.",
			Parameters:  json.RawMessage(`{"item_id":"*","content":"*"}`),
		},
		// --- Excel ---
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
			Name:        "Read from specific workbook",
			Description: "Locks the workbook; agent chooses the sheet and range to read.",
			Parameters:  json.RawMessage(`{"item_id":"WORKBOOK_ID","sheet_name":"*","range":"*"}`),
		},
		{
			ID:          "tpl_microsoft_excel_read_any",
			ActionType:  "microsoft.excel_read_range",
			Name:        "Read from any workbook",
			Description: "Agent can read from any Excel workbook the user has access to.",
			Parameters:  json.RawMessage(`{"item_id":"*","sheet_name":"*","range":"*"}`),
		},
		{
			ID:          "tpl_microsoft_excel_write_range",
			ActionType:  "microsoft.excel_write_range",
			Name:        "Write to specific workbook",
			Description: "Locks the workbook; agent chooses the sheet, range, and values to write.",
			Parameters:  json.RawMessage(`{"item_id":"WORKBOOK_ID","sheet_name":"*","range":"*","values":"*"}`),
		},
		{
			ID:          "tpl_microsoft_excel_append_rows",
			ActionType:  "microsoft.excel_append_rows",
			Name:        "Append to specific workbook",
			Description: "Locks the workbook; agent chooses the table and rows to append.",
			Parameters:  json.RawMessage(`{"item_id":"WORKBOOK_ID","table_name":"*","values":"*"}`),
		},
		{
			ID:          "tpl_microsoft_create_spreadsheet",
			ActionType:  "microsoft.create_spreadsheet",
			Name:        "Create spreadsheets",
			Description: "Agent can create new Excel spreadsheets in OneDrive.",
			Parameters:  json.RawMessage(`{"filename":"*","folder_path":"*"}`),
		},
		// --- Presentations ---
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
			ID:          "tpl_microsoft_create_presentation",
			ActionType:  "microsoft.create_presentation",
			Name:        "Create presentations",
			Description: "Agent can create new PowerPoint presentations in OneDrive.",
			Parameters:  json.RawMessage(`{"filename":"*","folder_path":"*"}`),
		},
		// --- Teams ---
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
			ID:          "tpl_microsoft_list_channel_messages",
			ActionType:  "microsoft.list_channel_messages",
			Name:        "Read channel messages",
			Description: "Agent can read messages from any Teams channel.",
			Parameters:  json.RawMessage(`{"team_id":"*","channel_id":"*","top":"*"}`),
		},
		{
			ID:          "tpl_microsoft_send_channel_message",
			ActionType:  "microsoft.send_channel_message",
			Name:        "Send channel messages",
			Description: "Agent can send messages to any Teams channel.",
			Parameters:  json.RawMessage(`{"team_id":"*","channel_id":"*","message":"*","reply_to_message_id":"*"}`),
		},
	}
}
