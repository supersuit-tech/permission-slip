// Package zoom implements the Zoom connector for the Permission Slip
// connector execution layer. It uses the Zoom REST API v2 with OAuth 2.0
// access tokens provided by the platform.
package zoom

import (
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
	defaultBaseURL     = "https://api.zoom.us/v2"
	defaultTimeout     = 30 * time.Second
	credKeyAccessToken = "access_token"

	// defaultRetryAfter is used when Zoom returns a 429 without a
	// Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the response body we'll read from Zoom APIs.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// ZoomConnector owns the shared HTTP client and base URL used by all
// Zoom actions. Actions hold a pointer back to the connector to access
// these shared resources.
type ZoomConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a ZoomConnector with sensible defaults.
func New() *ZoomConnector {
	return &ZoomConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a ZoomConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *ZoomConnector {
	return &ZoomConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "zoom", matching the connectors.id in the database.
func (c *ZoomConnector) ID() string { return "zoom" }

// Actions returns the registered action handlers keyed by action_type.
func (c *ZoomConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"zoom.list_meetings":           &listMeetingsAction{conn: c},
		"zoom.create_meeting":          &createMeetingAction{conn: c},
		"zoom.get_meeting":             &getMeetingAction{conn: c},
		"zoom.update_meeting":          &updateMeetingAction{conn: c},
		"zoom.delete_meeting":          &deleteMeetingAction{conn: c},
		"zoom.list_recordings":         &listRecordingsAction{conn: c},
		"zoom.get_meeting_participants": &getMeetingParticipantsAction{conn: c},
	}
}

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

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token.
func (c *ZoomConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// doJSON is the shared request lifecycle for Zoom API calls that send and
// receive JSON. It marshals reqBody as JSON, sends the request with OAuth
// bearer auth, handles rate limiting and timeouts, and unmarshals the
// response into respBody.
//
// For DELETE requests that return 204 No Content, pass nil for respBody.
func (c *ZoomConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Zoom API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Zoom API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Zoom API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Zoom API response",
			}
		}
	}

	return nil
}

// zoomAPIError represents the error response format from the Zoom API.
// Zoom returns {"code": <int>, "message": "<string>"} on errors.
type zoomAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// checkResponse maps HTTP status codes to typed connector errors.
// Zoom returns {"code": ..., "message": "..."} on errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract a Zoom API error message.
	var apiErr zoomAPIError
	msg := "Zoom API error"
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
		msg = apiErr.Message
	}

	switch {
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Zoom auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Zoom permission denied: %s", msg)}
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Zoom API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zoom API bad request: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zoom API not found: %s", msg)}
	case statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zoom API conflict: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Zoom API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
