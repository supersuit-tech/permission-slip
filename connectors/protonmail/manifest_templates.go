package protonmail

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func protonmailTemplates() []connectors.ManifestTemplate {
	return []connectors.ManifestTemplate{
		{
			ID:          "tpl_protonmail_reply",
			ActionType:  "protonmail.reply_email",
			Name:        "Reply to emails in your Proton Mail account",
			Description: "Agent can reply to existing emails on your behalf via your local Proton IMAP/SMTP proxy.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","in_reply_to_message_id":"*","body":"*"}`),
		},
		{
			ID:          "tpl_protonmail_send",
			ActionType:  "protonmail.send_email",
			Name:        "Send emails from your Proton Mail account",
			Description: "Agent can send emails on your behalf via your local Proton IMAP/SMTP proxy.",
			Parameters:  json.RawMessage(`{"to":"*","subject":"*","body":"*"}`),
		},
		{
			ID:          "tpl_protonmail_read_inbox",
			ActionType:  "protonmail.read_inbox",
			Name:        "Read recent inbox emails",
			Description: "Agent can read your recent inbox emails.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","limit":"*"}`),
		},
		{
			ID:          "tpl_protonmail_search",
			ActionType:  "protonmail.search_emails",
			Name:        "Search emails",
			Description: "Agent can search your emails by subject, sender, or date.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","limit":"*"}`),
		},
		{
			ID:          "tpl_protonmail_read_email",
			ActionType:  "protonmail.read_email",
			Name:        "Read a specific email",
			Description: "Agent can read the full content of a specific email.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","message_id":"*"}`),
		},
		{
			ID:          "tpl_protonmail_download_attachment",
			ActionType:  "protonmail.download_attachment",
			Name:        "Download an email attachment",
			Description: "Agent can download the content of an email attachment.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","message_id":"*","attachment_id":"*"}`),
		},
		{
			ID:          "tpl_protonmail_archive",
			ActionType:  "protonmail.archive_email",
			Name:        "Archive emails",
			Description: "Agent can move emails to the Archive folder.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","message_id":"*","message_ids":"*","include_thread":"*"}`),
		},
		{
			ID:          "tpl_protonmail_list_folders",
			ActionType:  "protonmail.list_folders",
			Name:        "List mailbox folders",
			Description: "Agent can list available mailbox folders.",
			Parameters:  json.RawMessage(`{}`),
		},
		{
			ID:          "tpl_protonmail_mark_read",
			ActionType:  "protonmail.mark_read",
			Name:        "Mark emails as read",
			Description: "Agent can mark emails as read.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","message_id":"*"}`),
		},
		{
			ID:          "tpl_protonmail_move",
			ActionType:  "protonmail.move_to_folder",
			Name:        "Move emails between folders",
			Description: "Agent can move emails to another folder.",
			Parameters:  json.RawMessage(`{"folder":"INBOX","target_folder":"*","message_id":"*"}`),
		},
	}
}
