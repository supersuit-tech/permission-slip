// Package twilio implements the Twilio connector for the Permission Slip
// connector execution layer. It uses the Twilio REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.twilio.com/2010-04-01"
	lookupBaseURL  = "https://lookups.twilio.com/v2"
	defaultTimeout = 30 * time.Second

	credKeyAccountSID = "account_sid"
	credKeyAuthToken  = "auth_token"

	// maxResponseBody is the maximum response body size we'll read from Twilio.
	// Prevents memory exhaustion from unexpectedly large responses.
	maxResponseBody = 1 << 20 // 1 MiB

	// Twilio Account SIDs start with "AC" and are 34 characters long.
	accountSIDPrefix = "AC"
	accountSIDLen    = 34

	// maxSMSBodyLen is the maximum length for an SMS message body.
	maxSMSBodyLen = 1600
)

// e164Pattern matches phone numbers in E.164 format: + followed by 1-15 digits.
var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// validateE164 validates that a phone number is in E.164 format and returns
// a clear error message for common mistakes.
func validateE164(field, value string) error {
	if !e164Pattern.MatchString(value) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("%s must be in E.164 format (e.g. +15551234567), got %q", field, value),
		}
	}
	return nil
}

// TwilioConnector owns the shared HTTP client and base URLs used by all
// Twilio actions. Actions hold a pointer back to the connector to access
// these shared resources.
type TwilioConnector struct {
	client    *http.Client
	baseURL   string
	lookupURL string
}

// New creates a TwilioConnector with sensible defaults (30s timeout,
// production Twilio API base URLs).
func New() *TwilioConnector {
	return &TwilioConnector{
		client:    &http.Client{Timeout: defaultTimeout},
		baseURL:   defaultBaseURL,
		lookupURL: lookupBaseURL,
	}
}

// newForTest creates a TwilioConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *TwilioConnector {
	return &TwilioConnector{
		client:    client,
		baseURL:   baseURL,
		lookupURL: baseURL,
	}
}

// ID returns "twilio", matching the connectors.id in the database.
func (c *TwilioConnector) ID() string { return "twilio" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *TwilioConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "twilio",
		Name:        "Twilio",
		Description: "Twilio integration for SMS, voice calls, and WhatsApp messaging",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "twilio.send_sms",
				Name:        "Send SMS",
				Description: "Send an SMS or MMS message",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "from", "body"],
					"properties": {
						"to": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Destination phone number in E.164 format (e.g. +15551234567)"
						},
						"from": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Twilio phone number to send from in E.164 format"
						},
						"body": {
							"type": "string",
							"maxLength": 1600,
							"description": "Message text (max 1600 characters)"
						},
						"media_url": {
							"type": "string",
							"format": "uri",
							"description": "URL of media to include (MMS, US/Canada only)"
						}
					}
				}`)),
			},
			{
				ActionType:  "twilio.send_whatsapp",
				Name:        "Send WhatsApp Message",
				Description: "Send a WhatsApp message via Twilio",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "from", "body"],
					"properties": {
						"to": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Destination phone number in E.164 format (e.g. +15551234567)"
						},
						"from": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Twilio WhatsApp-enabled number in E.164 format"
						},
						"body": {
							"type": "string",
							"description": "Message text"
						}
					}
				}`)),
			},
			{
				ActionType:  "twilio.initiate_call",
				Name:        "Initiate Call",
				Description: "Initiate an outbound voice call",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "from", "twiml"],
					"properties": {
						"to": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Destination phone number in E.164 format"
						},
						"from": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Twilio phone number to call from in E.164 format"
						},
						"twiml": {
							"type": "string",
							"description": "TwiML instructions for the call (e.g. <Response><Say>Hello</Say></Response>)"
						}
					}
				}`)),
			},
			{
				ActionType:  "twilio.get_message",
				Name:        "Get Message Status",
				Description: "Check the delivery status of a message",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["message_sid"],
					"properties": {
						"message_sid": {
							"type": "string",
							"pattern": "^(SM|MM)[0-9a-f]{32}$",
							"description": "The SID of the message to check (starts with SM or MM)"
						}
					}
				}`)),
			},
			{
				ActionType:  "twilio.get_call",
				Name:        "Get Call Status",
				Description: "Check the status and details of a call",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["call_sid"],
					"properties": {
						"call_sid": {
							"type": "string",
							"pattern": "^CA[0-9a-f]{32}$",
							"description": "The SID of the call to check (starts with CA)"
						}
					}
				}`)),
			},
			{
				ActionType:  "twilio.lookup_phone",
				Name:        "Phone Number Lookup",
				Description: "Look up information about a phone number",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["phone_number"],
					"properties": {
						"phone_number": {
							"type": "string",
							"pattern": "^\\+[1-9]\\d{1,14}$",
							"description": "Phone number to look up in E.164 format (e.g. +15551234567)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "twilio", AuthType: "basic", InstructionsURL: "https://www.twilio.com/docs/iam/api-keys"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_twilio_send_sms_any",
				ActionType:  "twilio.send_sms",
				Name:        "Send SMS freely",
				Description: "Agent can send SMS to any number from any Twilio number.",
				Parameters:  json.RawMessage(`{"to":"*","from":"*","body":"*"}`),
			},
			{
				ID:          "tpl_twilio_send_sms_from",
				ActionType:  "twilio.send_sms",
				Name:        "Send SMS from specific number",
				Description: "Locks the sending number. Agent chooses recipient and message.",
				Parameters:  json.RawMessage(`{"to":"*","from":"+15551234567","body":"*"}`),
			},
			{
				ID:          "tpl_twilio_send_whatsapp",
				ActionType:  "twilio.send_whatsapp",
				Name:        "Send WhatsApp messages",
				Description: "Agent can send WhatsApp messages to any number.",
				Parameters:  json.RawMessage(`{"to":"*","from":"*","body":"*"}`),
			},
			{
				ID:          "tpl_twilio_initiate_call",
				ActionType:  "twilio.initiate_call",
				Name:        "Make phone calls",
				Description: "Agent can initiate outbound voice calls.",
				Parameters:  json.RawMessage(`{"to":"*","from":"*","twiml":"*"}`),
			},
			{
				ID:          "tpl_twilio_get_message",
				ActionType:  "twilio.get_message",
				Name:        "Check message status",
				Description: "Agent can check delivery status of any message.",
				Parameters:  json.RawMessage(`{"message_sid":"*"}`),
			},
			{
				ID:          "tpl_twilio_get_call",
				ActionType:  "twilio.get_call",
				Name:        "Check call status",
				Description: "Agent can check status of any call.",
				Parameters:  json.RawMessage(`{"call_sid":"*"}`),
			},
			{
				ID:          "tpl_twilio_lookup_phone",
				ActionType:  "twilio.lookup_phone",
				Name:        "Look up phone numbers",
				Description: "Agent can look up information about any phone number.",
				Parameters:  json.RawMessage(`{"phone_number":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *TwilioConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"twilio.send_sms":       &sendSMSAction{conn: c},
		"twilio.send_whatsapp":  &sendWhatsAppAction{conn: c},
		"twilio.initiate_call":  &initiateCallAction{conn: c},
		"twilio.get_message":    &getMessageAction{conn: c},
		"twilio.get_call":       &getCallAction{conn: c},
		"twilio.lookup_phone":   &lookupPhoneAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// valid account_sid and auth_token.
func (c *TwilioConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	sid, ok := creds.Get(credKeyAccountSID)
	if !ok || sid == "" {
		return &connectors.ValidationError{Message: "missing required credential: account_sid"}
	}
	if len(sid) != accountSIDLen || !strings.HasPrefix(sid, accountSIDPrefix) {
		return &connectors.ValidationError{Message: "account_sid must start with \"AC\" and be 34 characters"}
	}

	token, ok := creds.Get(credKeyAuthToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: auth_token"}
	}
	return nil
}

// doForm sends a form-encoded POST request to the Twilio REST API.
// Twilio's API uses application/x-www-form-urlencoded for write operations.
func (c *TwilioConnector) doForm(ctx context.Context, creds connectors.Credentials, path string, formData url.Values, respBody any) error {
	sid, _ := creds.Get(credKeyAccountSID)
	token, _ := creds.Get(credKeyAuthToken)

	reqURL := c.baseURL + "/Accounts/" + url.PathEscape(sid) + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(sid, token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Twilio API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Twilio API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Twilio response: %v", err)}
		}
	}
	return nil
}

// doGet sends a GET request to the Twilio REST API.
func (c *TwilioConnector) doGet(ctx context.Context, creds connectors.Credentials, fullURL string, respBody any) error {
	sid, _ := creds.Get(credKeyAccountSID)
	token, _ := creds.Get(credKeyAuthToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(sid, token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Twilio API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Twilio API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Twilio response: %v", err)}
		}
	}
	return nil
}
