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
		Description: "Google integration for Gmail, Calendar, Slides, Chat, and Meet",
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
					"https://www.googleapis.com/auth/presentations",
					"https://www.googleapis.com/auth/chat.spaces.readonly",
					"https://www.googleapis.com/auth/chat.messages.create",
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
		},
	}
}
