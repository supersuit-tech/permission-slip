// Package google implements the Google connector for the Permission Slip
// connector execution layer. It uses Google REST APIs (Gmail, Calendar, Chat)
// with plain net/http and OAuth 2.0 access tokens provided by the platform.
package google

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
	defaultGmailBaseURL    = "https://gmail.googleapis.com"
	defaultCalendarBaseURL = "https://www.googleapis.com/calendar/v3"
	defaultChatBaseURL     = "https://chat.googleapis.com"
	defaultTimeout         = 30 * time.Second
	credKeyAccessToken     = "access_token"

	// defaultRetryAfter is used when the Google API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes is the maximum response body size we'll read from
	// Google APIs. This prevents OOM from unexpectedly large responses.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
)

// GoogleConnector owns the shared HTTP client and base URLs used by all
// Google actions. Actions hold a pointer back to the connector to access
// these shared resources.
type GoogleConnector struct {
	client          *http.Client
	gmailBaseURL    string
	calendarBaseURL string
	chatBaseURL     string
}

// New creates a GoogleConnector with sensible defaults.
func New() *GoogleConnector {
	return &GoogleConnector{
		client:          &http.Client{Timeout: defaultTimeout},
		gmailBaseURL:    defaultGmailBaseURL,
		calendarBaseURL: defaultCalendarBaseURL,
		chatBaseURL:     defaultChatBaseURL,
	}
}

// newForTest creates a GoogleConnector that points at a test server.
func newForTest(client *http.Client, gmailBaseURL, calendarBaseURL string) *GoogleConnector {
	return newForTestWithChat(client, gmailBaseURL, calendarBaseURL, "")
}

func newForTestWithChat(client *http.Client, gmailBaseURL, calendarBaseURL, chatBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:          client,
		gmailBaseURL:    gmailBaseURL,
		calendarBaseURL: calendarBaseURL,
		chatBaseURL:     chatBaseURL,
	}
}

// ID returns "google", matching the connectors.id in the database.
func (c *GoogleConnector) ID() string { return "google" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *GoogleConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "google",
		Name:        "Google",
		Description: "Google integration for Gmail, Calendar, Chat, and Meet",
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
				Parameters:  json.RawMessage(`{"page_size":"*"}`),
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

// Actions returns the registered action handlers keyed by action_type.
func (c *GoogleConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"google.send_email":            &sendEmailAction{conn: c},
		"google.list_emails":           &listEmailsAction{conn: c},
		"google.create_calendar_event": &createCalendarEventAction{conn: c},
		"google.list_calendar_events":  &listCalendarEventsAction{conn: c},
		"google.send_chat_message":     &sendChatMessageAction{conn: c},
		"google.list_chat_spaces":      &listChatSpacesAction{conn: c},
		"google.create_meeting":        &createMeetingAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token. Since tokens are provided by the platform's
// OAuth infrastructure, format validation is minimal.
func (c *GoogleConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	return nil
}

// doJSON is the shared request lifecycle for Google API calls that send and
// receive JSON. It marshals reqBody as JSON, sends the request with OAuth
// bearer auth, handles rate limiting and timeouts, and unmarshals the
// response into respBody.
func (c *GoogleConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, url string, reqBody, respBody any) error {
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
			return &connectors.TimeoutError{Message: fmt.Sprintf("Google API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Google API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Google API request failed: %v", err)}
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
				Message:    "failed to decode Google API response",
			}
		}
	}

	return nil
}

// checkResponse maps HTTP status codes to typed connector errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract a Google API error message.
	var apiErr struct {
		Error struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"error"`
	}
	msg := "Google API error"
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		msg = apiErr.Error.Message
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Google auth error: %s", msg)}
	case http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Google permission denied: %s", msg)}
	case http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Google API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Google API error (HTTP %d): %s", statusCode, msg),
		}
	}
}
