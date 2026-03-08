package docusign

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest for auto-seeding DB rows.
func (c *DocuSignConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "docusign",
		Name:        "DocuSign",
		Description: "DocuSign e-signature integration for creating, sending, and managing envelopes",
		LogoSVG:     logoSVG,
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
				Service:       "docusign",
				AuthType:      "oauth2",
				OAuthProvider: "docusign",
				OAuthScopes:   []string{"signature"},
			},
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
