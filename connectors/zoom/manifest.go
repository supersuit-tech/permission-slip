package zoom

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *ZoomConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "zoom",
		Name:        "Zoom",
		Description: "Zoom integration for meetings, recordings, and participants",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "zoom.list_meetings",
				Name:        "List Meetings",
				Description: "List meetings for the authenticated user filtered by type",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"type": {
							"type": "string",
							"enum": ["scheduled", "live", "upcoming"],
							"default": "upcoming",
							"description": "Meeting type filter: scheduled, live, or upcoming"
						},
						"page_size": {
							"type": "integer",
							"default": 30,
							"minimum": 1,
							"maximum": 300,
							"description": "Number of meetings to return per page (1-300, default 30)"
						}
					}
				}`)),
			},
			{
				ActionType:  "zoom.create_meeting",
				Name:        "Create Meeting",
				Description: "Schedule a new Zoom meeting and return the join URL",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["topic"],
					"properties": {
						"topic": {
							"type": "string",
							"description": "Meeting topic/title"
						},
						"type": {
							"type": "integer",
							"enum": [1, 2],
							"default": 2,
							"description": "Meeting type: 1 (instant) or 2 (scheduled)"
						},
						"start_time": {
							"type": "string",
							"description": "Start time in ISO 8601 format (e.g. '2024-01-15T09:00:00Z')"
						},
						"duration": {
							"type": "integer",
							"description": "Meeting duration in minutes"
						},
						"timezone": {
							"type": "string",
							"description": "Timezone (e.g. 'America/New_York')"
						},
						"agenda": {
							"type": "string",
							"description": "Meeting agenda/description"
						},
						"settings": {
							"type": "object",
							"properties": {
								"join_before_host": {
									"type": "boolean",
									"description": "Allow participants to join before host"
								},
								"waiting_room": {
									"type": "boolean",
									"description": "Enable waiting room"
								}
							},
							"description": "Meeting settings"
						}
					}
				}`)),
			},
			{
				ActionType:  "zoom.get_meeting",
				Name:        "Get Meeting Details",
				Description: "Get full details of a specific meeting including join URL, settings, and dial-in numbers",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["meeting_id"],
					"properties": {
						"meeting_id": {
							"type": "string",
							"description": "The meeting ID to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "zoom.update_meeting",
				Name:        "Update Meeting",
				Description: "Update an existing scheduled meeting — may notify participants of changes",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["meeting_id"],
					"properties": {
						"meeting_id": {
							"type": "string",
							"description": "The meeting ID to update"
						},
						"topic": {
							"type": "string",
							"description": "Updated meeting topic"
						},
						"start_time": {
							"type": "string",
							"description": "Updated start time in ISO 8601 format"
						},
						"duration": {
							"type": "integer",
							"description": "Updated duration in minutes"
						},
						"timezone": {
							"type": "string",
							"description": "Updated timezone"
						},
						"agenda": {
							"type": "string",
							"description": "Updated agenda"
						},
						"settings": {
							"type": "object",
							"properties": {
								"join_before_host": {
									"type": "boolean",
									"description": "Allow participants to join before host"
								},
								"waiting_room": {
									"type": "boolean",
									"description": "Enable waiting room"
								}
							},
							"description": "Updated meeting settings"
						}
					}
				}`)),
			},
			{
				ActionType:  "zoom.delete_meeting",
				Name:        "Delete Meeting",
				Description: "Delete/cancel a scheduled meeting — cancels for all participants",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["meeting_id"],
					"properties": {
						"meeting_id": {
							"type": "string",
							"description": "The meeting ID to delete"
						},
						"schedule_for_reminder": {
							"type": "boolean",
							"description": "Send a cancellation reminder to participants"
						}
					}
				}`)),
			},
			{
				ActionType:  "zoom.list_recordings",
				Name:        "List Recordings",
				Description: "List cloud recordings for the authenticated user within a date range",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["from", "to"],
					"properties": {
						"from": {
							"type": "string",
							"description": "Start date in YYYY-MM-DD format"
						},
						"to": {
							"type": "string",
							"description": "End date in YYYY-MM-DD format"
						},
						"page_size": {
							"type": "integer",
							"default": 30,
							"minimum": 1,
							"maximum": 300,
							"description": "Number of recordings to return per page (1-300, default 30)"
						}
					}
				}`)),
			},
			{
				ActionType:  "zoom.get_meeting_participants",
				Name:        "Get Meeting Participants",
				Description: "Get participant list for a past meeting (requires the meeting to have ended)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["meeting_id"],
					"properties": {
						"meeting_id": {
							"type": "string",
							"description": "The meeting ID to get participants for"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "zoom",
				AuthType:      "oauth2",
				OAuthProvider: "zoom",
				OAuthScopes: []string{
					"meeting:read",
					"meeting:write",
					"recording:read",
					"user:read",
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_zoom_list_meetings",
				ActionType:  "zoom.list_meetings",
				Name:        "List upcoming meetings",
				Description: "Agent can list upcoming meetings for the authenticated user.",
				Parameters:  json.RawMessage(`{"type":"*","page_size":"*"}`),
			},
			{
				ID:          "tpl_zoom_create_meeting",
				ActionType:  "zoom.create_meeting",
				Name:        "Schedule a Zoom meeting",
				Description: "Agent can schedule new Zoom meetings with any settings.",
				Parameters:  json.RawMessage(`{"topic":"*","type":"*","start_time":"*","duration":"*","timezone":"*","agenda":"*","settings":"*"}`),
			},
			{
				ID:          "tpl_zoom_create_meeting_30min",
				ActionType:  "zoom.create_meeting",
				Name:        "Schedule a 30-min Zoom call",
				Description: "Agent can schedule 30-minute Zoom meetings.",
				Parameters:  json.RawMessage(`{"topic":"*","type":2,"start_time":"*","duration":30,"timezone":"*","agenda":"*","settings":"*"}`),
			},
			{
				ID:          "tpl_zoom_get_meeting",
				ActionType:  "zoom.get_meeting",
				Name:        "View meeting details",
				Description: "Agent can view details of any meeting.",
				Parameters:  json.RawMessage(`{"meeting_id":"*"}`),
			},
			{
				ID:          "tpl_zoom_update_meeting",
				ActionType:  "zoom.update_meeting",
				Name:        "Update meetings",
				Description: "Agent can update any scheduled meeting.",
				Parameters:  json.RawMessage(`{"meeting_id":"*","topic":"*","start_time":"*","duration":"*","timezone":"*","agenda":"*","settings":"*"}`),
			},
			{
				ID:          "tpl_zoom_delete_meeting",
				ActionType:  "zoom.delete_meeting",
				Name:        "Cancel meetings",
				Description: "Agent can cancel any scheduled meeting.",
				Parameters:  json.RawMessage(`{"meeting_id":"*","schedule_for_reminder":"*"}`),
			},
			{
				ID:          "tpl_zoom_list_recordings",
				ActionType:  "zoom.list_recordings",
				Name:        "Find recordings from last week",
				Description: "Agent can search cloud recordings within a date range.",
				Parameters:  json.RawMessage(`{"from":"*","to":"*","page_size":"*"}`),
			},
			{
				ID:          "tpl_zoom_get_meeting_participants",
				ActionType:  "zoom.get_meeting_participants",
				Name:        "View meeting participants",
				Description: "Agent can view participant lists for past meetings.",
				Parameters:  json.RawMessage(`{"meeting_id":"*"}`),
			},
		},
	}
}
