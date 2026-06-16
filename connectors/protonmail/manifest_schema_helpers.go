package protonmail

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func uidMessageParametersSchema() json.RawMessage {
	return json.RawMessage(connectors.TrimIndent(`{
		"type": "object",
		"anyOf": [
			{"required": ["message_id"]},
			{"required": ["message_ids"]}
		],
		"properties": {
			"message_id": {
				"type": "integer",
				"minimum": 1,
				"description": "Stable IMAP UID of a single email (shorthand for message_ids with one item)"
			},
			"message_ids": {
				"type": "array",
				"items": {"type": "integer", "minimum": 1},
				"minItems": 1,
				"maxItems": 50,
				"description": "Stable IMAP UIDs of emails (batch). Combined unique count of message_id + message_ids must not exceed 50."
			},
			"folder": {
				"type": "string",
				"default": "INBOX",
				"description": "Mailbox folder containing the emails"
			}
		}
	}`))
}

func labelMessageParametersSchema() json.RawMessage {
	return json.RawMessage(connectors.TrimIndent(`{
		"type": "object",
		"required": ["label"],
		"anyOf": [
			{"required": ["message_id"]},
			{"required": ["message_ids"]}
		],
		"properties": {
			"message_id": {
				"type": "integer",
				"minimum": 1,
				"description": "Stable IMAP UID of a single email. By default the whole conversation is labeled (matching archive_email thread expansion). Set include_thread to false to label only this exact UID."
			},
			"message_ids": {
				"type": "array",
				"items": {"type": "integer", "minimum": 1},
				"minItems": 1,
				"maxItems": 50,
				"description": "Stable IMAP UIDs to label (batch). Combined unique count of message_id + message_ids must not exceed 50. By default each UID's whole conversation is labeled. Set include_thread to false to label only the exact UIDs listed."
			},
			"folder": {
				"type": "string",
				"default": "INBOX",
				"description": "Source mailbox folder containing the emails"
			},
			"label": {
				"type": "string",
				"description": "Proton label to apply or remove. Accepts a short name (e.g. Work) or full IMAP mailbox path under Labels/ (e.g. Labels/Work). Call protonmail.list_labels to discover valid labels."
			},
			"include_thread": {
				"type": "boolean",
				"default": true,
				"description": "When true (default), label every message in the conversation, not just the listed UIDs — matching archive_email thread expansion. Set to false to label only the exact UIDs provided."
			}
		}
	}`))
}

func moveToFolderParametersSchema() json.RawMessage {
	return json.RawMessage(connectors.TrimIndent(`{
		"type": "object",
		"required": ["target_folder"],
		"anyOf": [
			{"required": ["message_id"]},
			{"required": ["message_ids"]}
		],
		"properties": {
			"message_id": {
				"type": "integer",
				"minimum": 1,
				"description": "Stable IMAP UID of a single email (shorthand for message_ids with one item)"
			},
			"message_ids": {
				"type": "array",
				"items": {"type": "integer", "minimum": 1},
				"minItems": 1,
				"maxItems": 50,
				"description": "Stable IMAP UIDs of emails (batch). Combined unique count of message_id + message_ids must not exceed 50."
			},
			"folder": {
				"type": "string",
				"default": "INBOX",
				"description": "Source mailbox folder containing the emails"
			},
			"target_folder": {
				"type": "string",
				"description": "Destination mailbox folder"
			}
		}
	}`))
}
