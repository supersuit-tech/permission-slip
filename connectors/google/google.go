// Package google implements the Google connector for the Permission Slip
// connector execution layer. It uses Google REST APIs (Gmail, Calendar) with
// plain net/http and OAuth 2.0 access tokens provided by the platform.
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
	defaultSlidesBaseURL   = "https://slides.googleapis.com"
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
	slidesBaseURL   string
}

// New creates a GoogleConnector with sensible defaults.
func New() *GoogleConnector {
	return &GoogleConnector{
		client:          &http.Client{Timeout: defaultTimeout},
		gmailBaseURL:    defaultGmailBaseURL,
		calendarBaseURL: defaultCalendarBaseURL,
		slidesBaseURL:   defaultSlidesBaseURL,
	}
}

// newForTest creates a GoogleConnector that points at a test server.
func newForTest(client *http.Client, gmailBaseURL, calendarBaseURL, slidesBaseURL string) *GoogleConnector {
	return &GoogleConnector{
		client:          client,
		gmailBaseURL:    gmailBaseURL,
		calendarBaseURL: calendarBaseURL,
		slidesBaseURL:   slidesBaseURL,
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
		Description: "Google integration for Gmail, Calendar, and Slides",
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
							"enum": ["BLANK", "TITLE", "TITLE_AND_BODY"],
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
		"google.create_presentation":   &createPresentationAction{conn: c},
		"google.get_presentation":      &getPresentationAction{conn: c},
		"google.add_slide":             &addSlideAction{conn: c},
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
