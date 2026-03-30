package calendly

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *CalendlyConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "calendly",
		Name:        "Calendly",
		Description: "Calendly integration for scheduling, event types, and availability",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "calendly.list_event_types",
				Name:        "List Event Types",
				Description: "List available scheduling event types (e.g., \"30 min meeting\", \"1 hour consultation\")",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"user_uri": {
							"type": "string",
							"description": "The user URI to list event types for. If omitted, fetches automatically from /users/me. Pass this to avoid an extra API call if you already have the URI."
						},
						"active": {
							"type": "boolean",
							"description": "Filter by active status. If omitted, returns all event types."
						}
					}
				}`)),
			},
			{
				ActionType:  "calendly.create_scheduling_link",
				Name:        "Create Scheduling Link",
				Description: "Generate a single-use or reusable scheduling link for a specific event type",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_type_uri"],
					"properties": {
						"event_type_uri": {
							"type": "string",
							"description": "The URI of the event type to create a scheduling link for"
						},
						"max_event_count": {
							"type": "integer",
							"default": 1,
							"minimum": 1,
							"description": "Maximum number of events that can be scheduled using this link (default 1)"
						},
						"owner_type": {
							"type": "string",
							"enum": ["EventType"],
							"default": "EventType",
							"description": "The type of the owner of the scheduling link"
						}
					}
				}`)),
			},
			{
				ActionType:  "calendly.list_scheduled_events",
				Name:        "List Scheduled Events",
				Description: "List upcoming/past scheduled events with attendee details",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"user_uri": {
							"type": "string",
							"description": "The user URI to list events for. If omitted, fetches automatically from /users/me. Pass this to avoid an extra API call if you already have the URI."
						},
						"min_start_time": {
							"type": "string",
							"format": "date-time",
							"description": "Only return events starting at or after this time (ISO 8601 format)",
							"x-ui": {
								"label": "Start of range (min start time)",
								"datetime_range_pair": "max_start_time",
								"datetime_range_role": "lower"
							}
						},
						"max_start_time": {
							"type": "string",
							"format": "date-time",
							"description": "Only return events starting before this time (ISO 8601 format)",
							"x-ui": {
								"label": "End of range (max start time)",
								"datetime_range_pair": "min_start_time",
								"datetime_range_role": "upper"
							}
						},
						"status": {
							"type": "string",
							"enum": ["active", "canceled"],
							"description": "Filter by event status"
						},
						"count": {
							"type": "integer",
							"default": 20,
							"minimum": 1,
							"maximum": 100,
							"description": "Number of events to return (1-100, default 20)"
						}
					}
				}`)),
			},
			{
				ActionType:  "calendly.cancel_event",
				Name:        "Cancel Event",
				Description: "Cancel a scheduled event — sends a cancellation email to all participants",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_uuid"],
					"properties": {
						"event_uuid": {
							"type": "string",
							"description": "The UUID of the scheduled event to cancel"
						},
						"reason": {
							"type": "string",
							"description": "Cancellation reason text (sent to participants)"
						}
					}
				}`)),
			},
			{
				ActionType:  "calendly.get_event",
				Name:        "Get Event Details",
				Description: "Get full details of a specific scheduled event including attendees, location, and start/end times",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_uuid"],
					"properties": {
						"event_uuid": {
							"type": "string",
							"description": "The UUID of the scheduled event to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "calendly.list_available_times",
				Name:        "List Available Times",
				Description: "Check available time slots for a given event type and date range",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["event_type_uri", "start_time", "end_time"],
					"properties": {
						"event_type_uri": {
							"type": "string",
							"description": "The URI of the event type to check availability for"
						},
						"start_time": {
							"type": "string",
							"format": "date-time",
							"description": "Start of the time range to check (ISO 8601 format)",
							"x-ui": {
								"datetime_range_pair": "end_time",
								"datetime_range_role": "lower"
							}
						},
						"end_time": {
							"type": "string",
							"format": "date-time",
							"description": "End of the time range to check (ISO 8601 format)",
							"x-ui": {
								"datetime_range_pair": "start_time",
								"datetime_range_role": "upper"
							}
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "calendly_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "calendly",
			},
			{
				Service:         "calendly",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.calendly.com/how-to-authenticate-with-personal-access-tokens",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_calendly_list_event_types",
				ActionType:  "calendly.list_event_types",
				Name:        "List active event types",
				Description: "Agent can list active scheduling event types.",
				Parameters:  json.RawMessage(`{"user_uri":"*","active":"*"}`),
			},
			{
				ID:          "tpl_calendly_create_scheduling_link",
				ActionType:  "calendly.create_scheduling_link",
				Name:        "Share scheduling link for 30-min meeting",
				Description: "Agent can generate a single-use scheduling link for any event type.",
				Parameters:  json.RawMessage(`{"event_type_uri":"*","max_event_count":"*","owner_type":"*"}`),
			},
			{
				ID:          "tpl_calendly_list_scheduled_events",
				ActionType:  "calendly.list_scheduled_events",
				Name:        "List upcoming events",
				Description: "Agent can list upcoming scheduled events.",
				Parameters:  json.RawMessage(`{"user_uri":"*","min_start_time":"*","max_start_time":"*","status":"*","count":"*"}`),
			},
			{
				ID:          "tpl_calendly_cancel_event",
				ActionType:  "calendly.cancel_event",
				Name:        "Cancel a scheduled event",
				Description: "Agent can cancel scheduled events with a reason.",
				Parameters:  json.RawMessage(`{"event_uuid":"*","reason":"*"}`),
			},
			{
				ID:          "tpl_calendly_get_event",
				ActionType:  "calendly.get_event",
				Name:        "View event details",
				Description: "Agent can view details of any scheduled event.",
				Parameters:  json.RawMessage(`{"event_uuid":"*"}`),
			},
			{
				ID:          "tpl_calendly_list_available_times",
				ActionType:  "calendly.list_available_times",
				Name:        "Check my availability this week",
				Description: "Agent can check available time slots for any event type.",
				Parameters:  json.RawMessage(`{"event_type_uri":"*","start_time":"*","end_time":"*"}`),
			},
		},
	}
}
