// Package microsoft implements the Microsoft connector for the Permission Slip
// connector execution layer. It uses the Microsoft Graph API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package microsoft

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
func (c *MicrosoftConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "microsoft",
		Name:        "Microsoft",
		Description: "Microsoft 365 integration for email, calendar, and presentations via Microsoft Graph API",
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
				Parameters:  json.RawMessage(`{"subject":"*","start":"*","end":"*","time_zone":"*","body":"*","attendees":"*","location":"*"}`),
			},
			{
				ID:          "tpl_microsoft_list_events",
				ActionType:  "microsoft.list_calendar_events",
				Name:        "View calendar",
				Description: "Agent can view upcoming calendar events.",
				Parameters:  json.RawMessage(`{"top":"*"}`),
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
		"microsoft.create_presentation":   &createPresentationAction{conn: c},
		"microsoft.list_presentations":    &listPresentationsAction{conn: c},
		"microsoft.get_presentation":      &getPresentationAction{conn: c},
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

// doPutFileRequest uploads raw file bytes via PUT to a Microsoft Graph endpoint.
// Used for OneDrive file upload endpoints. Delegates to executeAndHandleResponse
// for shared response handling.
func (c *MicrosoftConnector) doPutFileRequest(ctx context.Context, path string, creds connectors.Credentials, fileBytes []byte, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(fileBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.openxmlformats-officedocument.presentationml.presentation")

	return c.executeAndHandleResponse(req, dest)
}

// doRequest is the shared request lifecycle for JSON-based Microsoft Graph actions.
// It handles JSON marshaling of the request body, auth, and delegates to
// executeAndHandleResponse for rate limiting, error mapping, and response parsing.
func (c *MicrosoftConnector) doRequest(ctx context.Context, method, path string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	var reqBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
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

	return c.executeAndHandleResponse(req, dest)
}

// executeAndHandleResponse executes an HTTP request and handles the response
// lifecycle shared by all Graph API calls: transport errors, rate limiting,
// response body reading with size limits, error mapping, and JSON decoding.
func (c *MicrosoftConnector) executeAndHandleResponse(req *http.Request, dest any) error {
	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Microsoft Graph API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Microsoft Graph API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Microsoft Graph API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Microsoft Graph API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapGraphError(resp.StatusCode, respBody)
	}

	// Some endpoints return 204 No Content (e.g. sendMail).
	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Microsoft Graph API response",
			}
		}
	}

	return nil
}
