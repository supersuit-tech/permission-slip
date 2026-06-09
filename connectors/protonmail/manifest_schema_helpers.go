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
