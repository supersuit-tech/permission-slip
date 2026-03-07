// Package docusign implements the DocuSign connector for the Permission Slip
// connector execution layer. It uses the DocuSign eSignature REST API v2.1
// with plain net/http (no third-party SDK).
package docusign

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://demo.docusign.net/restapi/v2.1"
	defaultTimeout = 30 * time.Second

	credKeyAccessToken = "access_token"
	credKeyAccountID   = "account_id"
	credKeyBaseURL     = "base_url" // optional — override for production or other regions

	// defaultRetryAfter is used when the DocuSign API returns a rate limit
	// response without a Retry-After header.
	defaultRetryAfter = 60 * time.Second

	// maxResponseBytes caps the DocuSign API response body at 50 MB to prevent
	// memory exhaustion (signed PDFs can be large).
	maxResponseBytes = 50 << 20 // 50 MB
)

// DocuSignConnector owns the shared HTTP client and base URL used by all
// DocuSign actions.
type DocuSignConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a DocuSignConnector with sensible defaults.
func New() *DocuSignConnector {
	return &DocuSignConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a DocuSignConnector pointing at a test server.
func newForTest(client *http.Client, baseURL string) *DocuSignConnector {
	return &DocuSignConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "docusign", matching the connectors.id in the database.
func (c *DocuSignConnector) ID() string { return "docusign" }

// Manifest returns the connector's metadata manifest for auto-seeding DB rows.
func (c *DocuSignConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "docusign",
		Name:        "DocuSign",
		Description: "DocuSign e-signature integration for creating, sending, and managing envelopes",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "docusign.create_envelope",
				Name:        "Create Envelope",
				Description: "Create a draft envelope from a template with recipients and field values",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["template_id", "recipients"],
					"properties": {
						"template_id": {
							"type": "string",
							"description": "DocuSign template ID to create the envelope from"
						},
						"email_subject": {
							"type": "string",
							"description": "Subject line for the signing notification email"
						},
						"email_body": {
							"type": "string",
							"description": "Body text for the signing notification email"
						},
						"recipients": {
							"type": "array",
							"description": "List of signers for this envelope",
							"items": {
								"type": "object",
								"required": ["email", "name", "role_name"],
								"properties": {
									"email": {
										"type": "string",
										"description": "Signer's email address"
									},
									"name": {
										"type": "string",
										"description": "Signer's full name"
									},
									"role_name": {
										"type": "string",
										"description": "Template role name to assign this signer to"
									}
								}
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "docusign.send_envelope",
				Name:        "Send Envelope",
				Description: "Send a draft envelope for signature — delivers legally binding documents to external parties",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["envelope_id"],
					"properties": {
						"envelope_id": {
							"type": "string",
							"description": "ID of the draft envelope to send"
						}
					}
				}`)),
			},
			{
				ActionType:  "docusign.check_status",
				Name:        "Check Envelope Status",
				Description: "Check the current status of an envelope and optionally see per-recipient signing progress",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["envelope_id"],
					"properties": {
						"envelope_id": {
							"type": "string",
							"description": "ID of the envelope to check"
						},
						"include_recipients": {
							"type": "boolean",
							"default": true,
							"description": "Include per-recipient signing status (who has signed, who hasn't)"
						}
					}
				}`)),
			},
			{
				ActionType:  "docusign.download_signed",
				Name:        "Download Signed Document",
				Description: "Download the signed document as a base64-encoded PDF",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["envelope_id"],
					"properties": {
						"envelope_id": {
							"type": "string",
							"description": "ID of the completed envelope to download"
						},
						"document_id": {
							"type": "string",
							"default": "combined",
							"description": "Document ID to download, or 'combined' for all documents merged into one PDF"
						}
					}
				}`)),
			},
			{
				ActionType:  "docusign.list_templates",
				Name:        "List Templates",
				Description: "List available envelope templates in the account",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"search_text": {
							"type": "string",
							"description": "Filter templates by name (case-insensitive substring match)"
						},
						"count": {
							"type": "integer",
							"default": 25,
							"description": "Max templates to return (1-100)"
						},
						"start_position": {
							"type": "integer",
							"default": 0,
							"description": "Starting position for pagination"
						}
					}
				}`)),
			},
			{
				ActionType:  "docusign.void_envelope",
				Name:        "Void Envelope",
				Description: "Void an in-progress envelope — cancels the signing process for all recipients",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["envelope_id", "void_reason"],
					"properties": {
						"envelope_id": {
							"type": "string",
							"description": "ID of the envelope to void"
						},
						"void_reason": {
							"type": "string",
							"description": "Reason for voiding the envelope (shown to recipients)"
						}
					}
				}`)),
			},
			{
				ActionType:  "docusign.update_recipients",
				Name:        "Update Recipients",
				Description: "Add or update recipients (signers) on a draft envelope",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["envelope_id", "signers"],
					"properties": {
						"envelope_id": {
							"type": "string",
							"description": "ID of the envelope to update"
						},
						"signers": {
							"type": "array",
							"description": "List of signers to add or update",
							"items": {
								"type": "object",
								"required": ["email", "name", "recipient_id", "routing_order"],
								"properties": {
									"email": {
										"type": "string",
										"description": "Signer's email address"
									},
									"name": {
										"type": "string",
										"description": "Signer's full name"
									},
									"recipient_id": {
										"type": "string",
										"description": "Unique recipient identifier within the envelope"
									},
									"routing_order": {
										"type": "string",
										"description": "Signing order (e.g. '1', '2')"
									}
								}
							}
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "docusign",
				AuthType:        "custom",
				InstructionsURL: "https://developers.docusign.com/docs/esign-rest-api/esign101/auth/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_docusign_create_from_template",
				ActionType:  "docusign.create_envelope",
				Name:        "Create envelope from template",
				Description: "Create a draft envelope from a specific template with recipients.",
				Parameters:  json.RawMessage(`{"template_id":"*","recipients":"*"}`),
			},
			{
				ID:          "tpl_docusign_send",
				ActionType:  "docusign.send_envelope",
				Name:        "Send envelope for signature",
				Description: "Send a draft envelope for signing.",
				Parameters:  json.RawMessage(`{"envelope_id":"*"}`),
			},
			{
				ID:          "tpl_docusign_check_status",
				ActionType:  "docusign.check_status",
				Name:        "Check envelope status",
				Description: "Check the status of any envelope.",
				Parameters:  json.RawMessage(`{"envelope_id":"*"}`),
			},
			{
				ID:          "tpl_docusign_list_templates",
				ActionType:  "docusign.list_templates",
				Name:        "List templates",
				Description: "Browse available envelope templates.",
				Parameters:  json.RawMessage(`{"search_text":"*","count":"*"}`),
			},
			{
				ID:          "tpl_docusign_void",
				ActionType:  "docusign.void_envelope",
				Name:        "Void an envelope",
				Description: "Cancel an in-progress envelope.",
				Parameters:  json.RawMessage(`{"envelope_id":"*","void_reason":"*"}`),
			},
			{
				ID:          "tpl_docusign_download",
				ActionType:  "docusign.download_signed",
				Name:        "Download signed document",
				Description: "Download a completed envelope as PDF.",
				Parameters:  json.RawMessage(`{"envelope_id":"*","document_id":"*"}`),
			},
			{
				ID:          "tpl_docusign_update_recipients",
				ActionType:  "docusign.update_recipients",
				Name:        "Update envelope recipients",
				Description: "Add or update signers on a draft envelope.",
				Parameters:  json.RawMessage(`{"envelope_id":"*","signers":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *DocuSignConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"docusign.create_envelope":   &createEnvelopeAction{conn: c},
		"docusign.send_envelope":     &sendEnvelopeAction{conn: c},
		"docusign.check_status":      &checkStatusAction{conn: c},
		"docusign.download_signed":   &downloadSignedAction{conn: c},
		"docusign.list_templates":    &listTemplatesAction{conn: c},
		"docusign.void_envelope":     &voidEnvelopeAction{conn: c},
		"docusign.update_recipients": &updateRecipientsAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty access_token and account_id.
func (c *DocuSignConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_token"}
	}
	accountID, ok := creds.Get(credKeyAccountID)
	if !ok || accountID == "" {
		return &connectors.ValidationError{Message: "missing required credential: account_id"}
	}
	return nil
}

// accountPath builds the account-scoped API path prefix with proper escaping.
func accountPath(accountID string) string {
	return "/accounts/" + url.PathEscape(accountID)
}

// resolveBaseURL returns the base URL to use for API requests. If the user
// has configured a base_url credential (for production or a specific region),
// that takes precedence over the connector's default (demo sandbox).
func (c *DocuSignConnector) resolveBaseURL(creds connectors.Credentials) string {
	if customURL, ok := creds.Get(credKeyBaseURL); ok && customURL != "" {
		return strings.TrimRight(customURL, "/")
	}
	return c.baseURL
}

// doJSON sends a JSON request to the DocuSign API and unmarshals the response.
func (c *DocuSignConnector) doJSON(ctx context.Context, method, path string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	baseURL := c.resolveBaseURL(creds)

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("DocuSign API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "DocuSign API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("DocuSign API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "DocuSign API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return &connectors.AuthError{Message: "DocuSign auth error: invalid or expired access token"}
	}

	if resp.StatusCode == http.StatusNotFound {
		var apiErr docuSignAPIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.ErrorCode != "" {
			return mapDocuSignError(resp.StatusCode, apiErr)
		}
		return &connectors.ValidationError{
			Message: fmt.Sprintf("DocuSign: resource not found (HTTP 404) — verify the envelope_id or document_id is correct"),
		}
	}

	if resp.StatusCode >= 400 {
		var apiErr docuSignAPIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.ErrorCode != "" {
			return mapDocuSignError(resp.StatusCode, apiErr)
		}
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("DocuSign API error (HTTP %d): %s", resp.StatusCode, truncateBody(respBody)),
		}
	}

	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode DocuSign API response",
			}
		}
	}

	return nil
}

// doRaw sends a request and returns the raw response body (for binary content like PDFs).
func (c *DocuSignConnector) doRaw(ctx context.Context, method, path string, creds connectors.Credentials) ([]byte, error) {
	token, ok := creds.Get(credKeyAccessToken)
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	baseURL := c.resolveBaseURL(creds)

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/pdf")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("DocuSign API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.TimeoutError{Message: "DocuSign API request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("DocuSign API request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return nil, &connectors.RateLimitError{
			Message:    "DocuSign API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, &connectors.AuthError{Message: "DocuSign auth error: invalid or expired access token"}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &connectors.ValidationError{
			Message: "DocuSign: resource not found (HTTP 404) — verify the envelope_id or document_id is correct",
		}
	}

	if resp.StatusCode >= 400 {
		return nil, &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("DocuSign API error (HTTP %d): %s", resp.StatusCode, truncateBody(body)),
		}
	}

	return body, nil
}

// maxErrorBodyLen is the maximum number of bytes from a response body to
// include in an error message. Prevents log bloat from large responses.
const maxErrorBodyLen = 512

// truncateBody returns the response body as a string, truncated to
// maxErrorBodyLen bytes to prevent oversized error messages.
func truncateBody(body []byte) string {
	if len(body) <= maxErrorBodyLen {
		return string(body)
	}
	return string(body[:maxErrorBodyLen]) + "... (truncated)"
}

// docuSignAPIError represents the standard DocuSign API error response.
type docuSignAPIError struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message"`
}

// mapDocuSignError converts a DocuSign API error to the appropriate connector
// error type. Provides actionable error messages for common DocuSign error codes
// so agents and users understand what went wrong and how to fix it.
func mapDocuSignError(statusCode int, apiErr docuSignAPIError) error {
	switch apiErr.ErrorCode {
	// Auth errors
	case "AUTHORIZATION_INVALID_TOKEN", "USER_AUTHENTICATION_FAILED",
		"ACCOUNT_NOT_AUTHORIZED", "USER_NOT_AUTHORIZED_FOR_ACCOUNT":
		return &connectors.AuthError{Message: fmt.Sprintf("DocuSign auth error: %s", apiErr.Message)}

	// Envelope state errors — include hints about what state is expected
	case "ENVELOPE_NOT_IN_CORRECT_STATE":
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign: envelope is not in the correct state for this operation — %s. Check the envelope status with docusign.check_status first.", apiErr.Message),
		}
	case "ENVELOPE_IS_INCOMPLETE":
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign: envelope is missing required information — %s. Ensure all recipients and document fields are filled.", apiErr.Message),
		}

	// Template errors
	case "TEMPLATE_NOT_FOUND":
		return &connectors.ValidationError{
			Message: "DocuSign: template not found. Use docusign.list_templates to browse available templates.",
		}

	// Recipient errors
	case "RECIPIENT_NOT_IN_SEQUENCE":
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign: recipient routing order conflict — %s. Ensure routing_order values are sequential.", apiErr.Message),
		}
	case "INVALID_EMAIL_ADDRESS_FOR_RECIPIENT":
		return &connectors.ValidationError{
			Message: fmt.Sprintf("DocuSign: invalid email address for recipient — %s", apiErr.Message),
		}

	// Resource not found
	case "ENVELOPE_DOES_NOT_EXIST":
		return &connectors.ValidationError{
			Message: "DocuSign: envelope not found. Verify the envelope_id is correct.",
		}

	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("DocuSign API error: %s — %s", apiErr.ErrorCode, apiErr.Message),
		}
	}
}
