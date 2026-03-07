// Package sendgrid implements the SendGrid connector for the Permission Slip
// connector execution layer. It uses the SendGrid v3 REST API with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package sendgrid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.sendgrid.com/v3"
	defaultTimeout = 30 * time.Second

	credKeyAPIKey = "api_key"

	// maxResponseBody is the maximum response body size we'll read from SendGrid.
	maxResponseBody = 1 << 20 // 1 MiB

	// minAPIKeyLen is the minimum length for a SendGrid API key.
	// SendGrid API keys are typically prefixed with "SG." and are fairly long.
	minAPIKeyLen = 10
)

// SendGridConnector owns the shared HTTP client and base URL used by all
// SendGrid actions. Actions hold a pointer back to the connector to access
// these shared resources.
type SendGridConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a SendGridConnector with sensible defaults (30s timeout,
// production SendGrid API base URL).
func New() *SendGridConnector {
	return &SendGridConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a SendGridConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *SendGridConnector {
	return &SendGridConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "sendgrid", matching the connectors.id in the database.
func (c *SendGridConnector) ID() string { return "sendgrid" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *SendGridConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "sendgrid",
		Name:        "SendGrid",
		Description: "SendGrid integration for email marketing — campaigns, subscriber lists, templates, and analytics",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "sendgrid.send_campaign",
				Name:        "Send Email Campaign",
				Description: "Send a single send email campaign to a list of recipients. WARNING: This immediately sends to all contacts in the specified lists.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "subject", "list_ids", "sender_id"],
					"properties": {
						"name": {
							"type": "string",
							"maxLength": 100,
							"description": "Internal name for this campaign (e.g. 'March 2026 Newsletter')"
						},
						"subject": {
							"type": "string",
							"maxLength": 998,
							"description": "Email subject line seen by recipients"
						},
						"html_content": {
							"type": "string",
							"description": "HTML body of the email. At least one of html_content or plain_content is required."
						},
						"plain_content": {
							"type": "string",
							"description": "Plain text body of the email. At least one of html_content or plain_content is required."
						},
						"list_ids": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"description": "Contact list IDs to send to. Use sendgrid.list_lists to find available list IDs."
						},
						"sender_id": {
							"type": "integer",
							"description": "Verified sender identity ID. Use sendgrid.list_senders to find your sender ID."
						},
						"suppression_group_id": {
							"type": "integer",
							"description": "Unsubscribe group ID for managing opt-outs (optional but recommended)"
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.schedule_campaign",
				Name:        "Schedule Email Campaign",
				Description: "Schedule a single send email campaign for future delivery. The campaign can be cancelled before the scheduled time.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "subject", "list_ids", "sender_id", "send_at"],
					"properties": {
						"name": {
							"type": "string",
							"maxLength": 100,
							"description": "Internal name for this campaign (e.g. 'April Product Launch')"
						},
						"subject": {
							"type": "string",
							"maxLength": 998,
							"description": "Email subject line seen by recipients"
						},
						"html_content": {
							"type": "string",
							"description": "HTML body of the email. At least one of html_content or plain_content is required."
						},
						"plain_content": {
							"type": "string",
							"description": "Plain text body of the email. At least one of html_content or plain_content is required."
						},
						"list_ids": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"description": "Contact list IDs to send to. Use sendgrid.list_lists to find available list IDs."
						},
						"sender_id": {
							"type": "integer",
							"description": "Verified sender identity ID. Use sendgrid.list_senders to find your sender ID."
						},
						"send_at": {
							"type": "string",
							"format": "date-time",
							"description": "ISO 8601 timestamp for when to send (must be in the future, e.g. 2026-03-15T10:00:00Z)"
						},
						"suppression_group_id": {
							"type": "integer",
							"description": "Unsubscribe group ID for managing opt-outs (optional but recommended)"
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.add_to_list",
				Name:        "Add Subscriber to List",
				Description: "Add a contact to a SendGrid contact list. The operation is asynchronous — the returned job_id can be used to track import progress.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["list_id", "email"],
					"properties": {
						"list_id": {
							"type": "string",
							"description": "Contact list ID. Use sendgrid.list_lists to find available list IDs."
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Subscriber email address"
						},
						"first_name": {
							"type": "string",
							"description": "Subscriber first name (optional)"
						},
						"last_name": {
							"type": "string",
							"description": "Subscriber last name (optional)"
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.remove_from_list",
				Name:        "Remove Subscriber from List",
				Description: "Remove a contact from a SendGrid contact list. This only removes the list association — the contact itself is not deleted.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["list_id", "contact_id"],
					"properties": {
						"list_id": {
							"type": "string",
							"description": "Contact list ID. Use sendgrid.list_lists to find available list IDs."
						},
						"contact_id": {
							"type": "string",
							"description": "Contact ID to remove from the list"
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.create_template",
				Name:        "Create Email Template",
				Description: "Create a dynamic transactional email template. After creating, add versions with HTML content via the SendGrid dashboard.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"maxLength": 100,
							"description": "Template name (e.g. 'Welcome Email', 'Order Confirmation')"
						},
						"generation": {
							"type": "string",
							"enum": ["legacy", "dynamic"],
							"description": "Template generation — use 'dynamic' for Handlebars support (default: dynamic)"
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.get_campaign_stats",
				Name:        "Get Campaign Stats",
				Description: "Get analytics for a single send campaign including delivery, open, click, bounce, and unsubscribe metrics",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["singlesend_id"],
					"properties": {
						"singlesend_id": {
							"type": "string",
							"description": "Single send campaign ID (returned by send_campaign or schedule_campaign)"
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.list_segments",
				Name:        "List Segments",
				Description: "List all contact segments in the account with subscriber counts",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "sendgrid.list_senders",
				Name:        "List Verified Senders",
				Description: "List all verified sender identities — use this to find sender_id values needed for campaigns",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "sendgrid.list_lists",
				Name:        "List Contact Lists",
				Description: "List all contact lists with subscriber counts — use this to find list_id values for campaigns and subscriber management",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "sendgrid",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.sendgrid.com/ui/account-and-settings/api-keys",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_sendgrid_send_campaign",
				ActionType:  "sendgrid.send_campaign",
				Name:        "Send email campaign",
				Description: "Agent can create and send email campaigns to any list. High risk — consider using the locked-list template instead.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","html_content":"*","plain_content":"*","list_ids":"*","sender_id":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_send_to_list",
				ActionType:  "sendgrid.send_campaign",
				Name:        "Send campaign to specific list",
				Description: "Locks the recipient list and sender — agent can only customize email content. Safer than the unrestricted template.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","html_content":"*","plain_content":"*","list_ids":["YOUR_LIST_ID"],"sender_id":0}`),
			},
			{
				ID:          "tpl_sendgrid_schedule_campaign",
				ActionType:  "sendgrid.schedule_campaign",
				Name:        "Schedule email campaign",
				Description: "Agent can create and schedule email campaigns for future delivery.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","html_content":"*","plain_content":"*","list_ids":"*","sender_id":"*","send_at":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_add_subscriber",
				ActionType:  "sendgrid.add_to_list",
				Name:        "Add subscriber to list",
				Description: "Agent can add contacts to any list.",
				Parameters:  json.RawMessage(`{"list_id":"*","email":"*","first_name":"*","last_name":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_add_to_specific_list",
				ActionType:  "sendgrid.add_to_list",
				Name:        "Add subscriber to specific list",
				Description: "Locks the target list — agent can only add contacts to this specific list.",
				Parameters:  json.RawMessage(`{"list_id":"YOUR_LIST_ID","email":"*","first_name":"*","last_name":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_remove_subscriber",
				ActionType:  "sendgrid.remove_from_list",
				Name:        "Remove subscriber from list",
				Description: "Agent can remove contacts from any list.",
				Parameters:  json.RawMessage(`{"list_id":"*","contact_id":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_create_template",
				ActionType:  "sendgrid.create_template",
				Name:        "Create email templates",
				Description: "Agent can create new email templates.",
				Parameters:  json.RawMessage(`{"name":"*","generation":"dynamic"}`),
			},
			{
				ID:          "tpl_sendgrid_get_stats",
				ActionType:  "sendgrid.get_campaign_stats",
				Name:        "View campaign analytics",
				Description: "Agent can check campaign stats like opens, clicks, and bounces.",
				Parameters:  json.RawMessage(`{"singlesend_id":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_list_segments",
				ActionType:  "sendgrid.list_segments",
				Name:        "List contact segments",
				Description: "Agent can list all contact segments.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_sendgrid_list_senders",
				ActionType:  "sendgrid.list_senders",
				Name:        "List verified senders",
				Description: "Agent can list all verified sender identities to find sender IDs.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_sendgrid_list_lists",
				ActionType:  "sendgrid.list_lists",
				Name:        "List contact lists",
				Description: "Agent can list all contact lists to find list IDs.",
				Parameters:  json.RawMessage(`{}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *SendGridConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"sendgrid.send_campaign":      &sendCampaignAction{conn: c},
		"sendgrid.schedule_campaign":  &scheduleCampaignAction{conn: c},
		"sendgrid.add_to_list":        &addToListAction{conn: c},
		"sendgrid.remove_from_list":   &removeFromListAction{conn: c},
		"sendgrid.create_template":    &createTemplateAction{conn: c},
		"sendgrid.get_campaign_stats": &getCampaignStatsAction{conn: c},
		"sendgrid.list_segments":      &listSegmentsAction{conn: c},
		"sendgrid.list_senders":       &listSendersAction{conn: c},
		"sendgrid.list_lists":         &listListsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// valid SendGrid API key.
func (c *SendGridConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get(credKeyAPIKey)
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	if len(key) < minAPIKeyLen {
		return &connectors.ValidationError{Message: fmt.Sprintf("api_key is too short (minimum %d characters)", minAPIKeyLen)}
	}
	return nil
}

// doJSON sends a JSON-encoded request to the SendGrid API and decodes
// the response. SendGrid's v3 API uses application/json throughout.
func (c *SendGridConnector) doJSON(ctx context.Context, creds connectors.Credentials, method, path string, body any, respBody any) error {
	apiKey, _ := creds.Get(credKeyAPIKey)

	reqURL := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("SendGrid API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("SendGrid API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing SendGrid response: %v", err)}
		}
	}
	return nil
}
