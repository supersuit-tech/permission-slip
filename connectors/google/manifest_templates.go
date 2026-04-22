package google

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// googleTemplates returns configuration templates for common Google Workspace
// use cases. Templates cover Gmail, Calendar, Drive, Docs, Sheets, Slides,
// Chat, and Meet — from read-only inbox access to scoped write operations.
func googleTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		// --- Gmail read ---
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
			ID:          "tpl_google_list_emails_from_domain",
			ActionType:  "google.list_emails",
			Name:        "List emails from a domain",
			Description: "Agent can list emails from senders matching a specific domain. Set the domain pattern before applying.",
			Parameters:  json.RawMessage(`{"query":"from:*@example.com","max_results":"*"}`),
		},
		{
			ID:          "tpl_google_read_email",
			ActionType:  "google.read_email",
			Name:        "Read any email",
			Description: "Agent can read the full content of any email by message ID.",
			Parameters:  json.RawMessage(`{"message_id":"*"}`),
		},
		// --- Gmail write ---
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
			ID:          "tpl_google_send_email_to_domain",
			ActionType:  "google.send_email",
			Name:        "Send email to a domain",
			Description: "Agent can only send emails to recipients matching a specific domain pattern. Set the domain before applying.",
			Parameters:  json.RawMessage(`{"to":{"$pattern":"*@example.com"},"subject":"*","body":"*"}`),
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
			Description: "Agent can archive Gmail threads (removes from inbox; still accessible via search and All Mail).",
			Parameters:  json.RawMessage(`{"thread_id":"*"}`),
		},
		// --- Calendar read ---
		{
			ID:          "tpl_google_list_calendar_events",
			ActionType:  "google.list_calendar_events",
			Name:        "List calendar events",
			Description: "Agent can list upcoming events from any calendar.",
			Parameters:  json.RawMessage(`{"calendar_id":"*","max_results":"*","time_min":"*","time_max":"*"}`),
		},
		// --- Calendar write ---
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
		// --- Drive read ---
		{
			ID:          "tpl_google_list_drive_files",
			ActionType:  "google.list_drive_files",
			Name:        "Browse Drive files",
			Description: "Agent can list and search files in Google Drive.",
			Parameters:  json.RawMessage(`{"query":"*","max_results":"*","folder_id":"*","order_by":"*"}`),
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
		// --- Drive write ---
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
			ID:          "tpl_google_create_drive_folder",
			ActionType:  "google.create_drive_folder",
			Name:        "Create Drive folders",
			Description: "Agent can create folders anywhere in Google Drive.",
			Parameters:  json.RawMessage(`{"name":"*","parent_id":"*"}`),
		},
		{
			ID:          "tpl_google_delete_drive_file",
			ActionType:  "google.delete_drive_file",
			Name:        "Trash Drive files",
			Description: "Agent can move files to trash in Google Drive.",
			Parameters:  json.RawMessage(`{"file_id":"*"}`),
		},
		// --- Docs ---
		{
			ID:          "tpl_google_list_documents",
			ActionType:  "google.list_documents",
			Name:        "Search documents",
			Description: "Agent can search and list Google Docs from Drive.",
			Parameters:  json.RawMessage(`{"query":"*","max_results":"*"}`),
		},
		{
			ID:          "tpl_google_get_document",
			ActionType:  "google.get_document",
			Name:        "Read any document",
			Description: "Agent can read the content of any Google Doc by ID.",
			Parameters:  json.RawMessage(`{"document_id":"*"}`),
		},
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
			ID:          "tpl_google_update_document",
			ActionType:  "google.update_document",
			Name:        "Edit any document",
			Description: "Agent can insert or append text to any Google Doc.",
			Parameters:  json.RawMessage(`{"document_id":"*","content":"*","index":"*"}`),
		},
		// --- Sheets ---
		{
			ID:          "tpl_google_sheets_list",
			ActionType:  "google.sheets_list_sheets",
			Name:        "List worksheets in any spreadsheet",
			Description: "Agent can list worksheets in any spreadsheet.",
			Parameters:  json.RawMessage(`{"spreadsheet_id":"*"}`),
		},
		{
			ID:          "tpl_google_sheets_read_range",
			ActionType:  "google.sheets_read_range",
			Name:        "Read from specific spreadsheet",
			Description: "Locks the spreadsheet; agent chooses the range to read.",
			Parameters:  json.RawMessage(`{"spreadsheet_id":"SPREADSHEET_ID","range":"*"}`),
		},
		{
			ID:          "tpl_google_sheets_read_any",
			ActionType:  "google.sheets_read_range",
			Name:        "Read from any spreadsheet",
			Description: "Agent can read from any spreadsheet and any range.",
			Parameters:  json.RawMessage(`{"spreadsheet_id":"*","range":"*"}`),
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
		// --- Slides ---
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
			Parameters:  json.RawMessage(`{"presentation_id":"*","layout":"*","insertion_index":"*","title":"*"}`),
		},
		// --- Chat ---
		{
			ID:          "tpl_google_list_chat_spaces",
			ActionType:  "google.list_chat_spaces",
			Name:        "List chat spaces",
			Description: "Agent can list Google Chat spaces accessible to the user.",
			Parameters:  json.RawMessage(`{"page_size":"*","filter":"*"}`),
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
	}
}
