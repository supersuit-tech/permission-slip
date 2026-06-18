package imessage

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

//go:embed logo.svg
var logoSVG string

const handleSchema = `{
	"type": "object",
	"required": ["type", "value"],
	"properties": {
		"type": {
			"type": "string",
			"enum": ["phone", "email"],
			"description": "Handle type: phone (E.164) or email (iMessage email address)"
		},
		"value": {
			"type": "string",
			"description": "Phone number in E.164 format or iMessage email address"
		}
	}
}`

func (c *IMessageConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "imessage",
		Name:        "iMessage",
		Description: "Read and send Messages.app conversations (iMessage and SMS) via openclaw/imsg on a Mac. Requires the same Apple ID on Mac and iPhone, imsg (brew install steipete/tap/imsg), Full Disk Access for reads, and Automation permission for sends.",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "imessage.list_chats",
				OperationType:   "read",
				Name:            "List Chats",
				Description:     "List recent iMessage and SMS conversations",
				RiskLevel:       "low",
				DisplayTemplate: "List {{limit:count}} recent chats",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 20,
							"description": "Maximum number of chats to return"
						}
					}
				}`)),
			},
			{
				ActionType:      "imessage.get_chat",
				OperationType:   "read",
				Name:            "Get Chat",
				Description:     "Get a single conversation with its participant list",
				RiskLevel:       "low",
				DisplayTemplate: "Get chat {{chat_id}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["chat_id"],
					"properties": {
						"chat_id": {
							"type": "integer",
							"minimum": 1,
							"description": "Chat ID from list_chats"
						}
					}
				}`)),
			},
			{
				ActionType:      "imessage.read_history",
				OperationType:   "read",
				Name:            "Read Message History",
				Description:     "Read messages in a chat, with optional incremental cursor for polling",
				RiskLevel:       "low",
				DisplayTemplate: "Read history for chat {{chat_id}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["chat_id"],
					"properties": {
						"chat_id": {
							"type": "integer",
							"minimum": 1,
							"description": "Chat ID from list_chats"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 200,
							"default": 50,
							"description": "Maximum number of messages to return"
						},
						"since_guid": {
							"type": "string",
							"description": "Return only messages newer than this message GUID (for deduplicated polling)"
						},
						"since_rowid": {
							"type": "integer",
							"minimum": 1,
							"description": "Return only messages with row ID greater than this value"
						},
						"attachments": {
							"type": "boolean",
							"default": false,
							"description": "Include attachment metadata in results"
						},
						"start": {
							"type": "string",
							"format": "date-time",
							"description": "Only messages on or after this ISO 8601 timestamp"
						},
						"end": {
							"type": "string",
							"format": "date-time",
							"description": "Only messages before this ISO 8601 timestamp"
						}
					}
				}`)),
			},
			{
				ActionType:      "imessage.search",
				OperationType:   "read",
				Name:            "Search Messages",
				Description:     "Search local message history by text",
				RiskLevel:       "low",
				DisplayTemplate: "Search messages for {{query}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Text to search for"
						},
						"match": {
							"type": "string",
							"enum": ["contains", "exact"],
							"default": "contains",
							"description": "Match mode: substring or exact"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 200,
							"default": 50,
							"description": "Maximum number of results"
						}
					}
				}`)),
			},
			{
				ActionType:      "imessage.send_message",
				OperationType:   "write",
				Name:            "Send Message",
				Description:     "Send an iMessage or SMS via Messages.app (approval required). Defaults to auto service with SMS fallback.",
				RiskLevel:       "high",
				DisplayTemplate: "Send message",
				Preview: &connectors.ActionPreview{
					Layout: "message",
					Fields: map[string]string{"to": "to", "body": "text"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"to": {
							"type": "array",
							"items": ` + handleSchema + `,
							"minItems": 1,
							"description": "Recipient handle(s) for a direct message. Use chat_id/chat_guid for group chats."
						},
						"from": {
							"type": "array",
							"items": ` + handleSchema + `,
							"description": "Sending account constraint metadata (resolved from imsg account; not sent to Messages.app)"
						},
						"chat_id": {
							"type": "integer",
							"minimum": 1,
							"description": "Existing chat ID (preferred for group chats)"
						},
						"chat_identifier": {
							"type": "string",
							"description": "Existing chat identifier"
						},
						"chat_guid": {
							"type": "string",
							"description": "Existing chat GUID (e.g. iMessage;+;chat0000...)"
						},
						"text": {
							"type": "string",
							"description": "Message text"
						},
						"file": {
							"type": "string",
							"description": "Local file path to attach"
						},
						"service": {
							"type": "string",
							"enum": ["imessage", "sms", "auto"],
							"default": "auto",
							"description": "Delivery service. Default auto picks iMessage or SMS based on the chat; SMS fallback applies for new recipients unless no_sms_fallback is true."
						},
						"no_sms_fallback": {
							"type": "boolean",
							"default": false,
							"description": "When true, do not fall back to SMS when iMessage is unavailable (opt-in strict iMessage-only)"
						},
						"retry_guid": {
							"type": "string",
							"description": "Optional message GUID from a prior send attempt. When set, skips sending if that message already reached sent/delivered state (idempotent retry)."
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "imessage",
				AuthType:        "custom",
				InstructionsURL: "https://github.com/openclaw/imsg",
				Fields: []connectors.ManifestCredentialField{
					{
						Key:         credKeyCLIPath,
						Label:       "imsg path",
						Placeholder: "imsg",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
						HelpText:    "Path to the imsg binary or a wrapper script. Defaults to imsg on PATH.",
					},
					{
						Key:         credKeyRemoteHost,
						Label:       "Remote SSH host",
						Placeholder: "messages-mac",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
						HelpText:    "Optional SSH host when Permission Slip runs on Linux and imsg runs on a separate Mac (e.g. ssh -T messages-mac imsg).",
					},
				},
			},
		},
		Templates: imessageTemplates(),
	}
}

func imessageTemplates() []connectors.ManifestTemplate {
	handleArray := `[{"type":"phone","value":"*"}]`
	return []connectors.ManifestTemplate{
		{
			ID:          "tpl_imessage_read_any",
			ActionType:  "imessage.read_history",
			Name:        "Read any chat history",
			Description: "Agent can read message history from any chat.",
			Parameters:  json.RawMessage(`{"chat_id":"*"}`),
		},
		{
			ID:          "tpl_imessage_search_any",
			ActionType:  "imessage.search",
			Name:        "Search all messages",
			Description: "Agent can search local message history.",
			Parameters:  json.RawMessage(`{"query":"*","match":"*"}`),
		},
		{
			ID:          "tpl_imessage_send_any",
			ActionType:  "imessage.send_message",
			Name:        "Send messages freely",
			Description: "Agent can send to any recipient from any signed-in account (auto service with SMS fallback).",
			Parameters:  json.RawMessage(`{"to":` + handleArray + `,"from":` + handleArray + `,"text":"*","service":"auto","no_sms_fallback":false}`),
		},
		{
			ID:          "tpl_imessage_send_to_contact",
			ActionType:  "imessage.send_message",
			Name:        "Send to specific contact",
			Description: "Locks the recipient handle. Agent chooses the message text.",
			Parameters:  json.RawMessage(`{"to":[{"type":"phone","value":"+15555550123"}],"from":"*","text":"*","service":"auto","no_sms_fallback":false}`),
		},
		{
			ID:          "tpl_imessage_send_imessage_only",
			ActionType:  "imessage.send_message",
			Name:        "Send iMessage only (no SMS fallback)",
			Description: "Agent can send iMessages but SMS fallback is disabled.",
			Parameters:  json.RawMessage(`{"to":` + handleArray + `,"from":` + handleArray + `,"text":"*","service":"imessage","no_sms_fallback":true}`),
		},
	}
}

func ptrBool(b bool) *bool { return &b }

var _ connectors.ManifestProvider = (*IMessageConnector)(nil)
