package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// archiveMailbox is the IMAP folder that Proton Mail Bridge exposes for
// archived messages. Proton Bridge maps the "Archive" label to this folder.
const archiveMailbox = "Archive"

// archiveEmailAction moves one or more emails to the Archive folder via IMAP
// MOVE (RFC 6851). Proton Mail Bridge supports MOVE natively.
type archiveEmailAction struct {
	conn *ProtonMailConnector
}

// archiveEmailRaw handles flexible JSON input: accepts either a single integer
// for message_id or an array for message_ids, so callers can archive one email
// without wrapping it in an array.
type archiveEmailRaw struct {
	MessageID  *uint32         `json:"message_id,omitempty"`
	MessageIDs json.RawMessage `json:"message_ids,omitempty"`
	Folder     string          `json:"folder"`
}

type archiveEmailParams struct {
	MessageIDs []uint32 `json:"-"`
	Folder     string   `json:"folder"`
}

// parseArchiveParams normalizes the flexible input into archiveEmailParams.
// Accepts "message_id": 5 (single), "message_ids": [1,2,3] (batch), or both
// (merged). Deduplication happens later in validate().
func parseArchiveParams(raw []byte) (*archiveEmailParams, error) {
	var r archiveEmailRaw
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	params := &archiveEmailParams{Folder: r.Folder}

	// Prefer message_ids array if provided.
	if len(r.MessageIDs) > 0 && string(r.MessageIDs) != "null" {
		if err := json.Unmarshal(r.MessageIDs, &params.MessageIDs); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid message_ids: %v", err)}
		}
	}

	// Also append message_id if provided; allows combining both fields.
	if r.MessageID != nil {
		params.MessageIDs = append(params.MessageIDs, *r.MessageID)
	}

	return params, nil
}

func (p *archiveEmailParams) validate() error {
	if len(p.MessageIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: provide message_id (single) or message_ids (array)"}
	}

	// Deduplicate: callers may accidentally pass the same ID twice.
	p.MessageIDs = deduplicateUint32(p.MessageIDs)

	if len(p.MessageIDs) > maxLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("too many message_ids: maximum is %d", maxLimit)}
	}
	for _, id := range p.MessageIDs {
		if id == 0 {
			return &connectors.ValidationError{Message: "message_ids must not contain zero values"}
		}
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	if strings.EqualFold(p.Folder, archiveMailbox) {
		return &connectors.ValidationError{Message: "cannot archive emails that are already in the Archive folder"}
	}
	return nil
}

// deduplicateUint32 returns a new slice with duplicate values removed,
// preserving the original order.
func deduplicateUint32(ids []uint32) []uint32 {
	seen := make(map[uint32]struct{}, len(ids))
	out := make([]uint32, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func (a *archiveEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseArchiveParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	session, err := connectIMAP(req.Credentials, a.conn.timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	// Open the source folder in read-write mode so we can move messages.
	mboxData, err := session.selectMailboxReadWrite(params.Folder)
	if err != nil {
		return nil, err
	}

	// Best-effort bounds check against the message count at SELECT time.
	// Concurrent EXPUNGE responses can make this stale, but it catches
	// obviously invalid IDs before hitting the server.
	for _, id := range params.MessageIDs {
		if id > mboxData.NumMessages {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("message_id %d not found (mailbox has %d messages)", id, mboxData.NumMessages),
			}
		}
	}

	var seqSet imap.SeqSet
	for _, id := range params.MessageIDs {
		seqSet.AddNum(id)
	}

	// MOVE messages to the Archive folder (RFC 6851).
	moveCmd := session.client.Move(seqSet, archiveMailbox)
	if _, err := moveCmd.Wait(); err != nil {
		imapErr := mapIMAPError(err)
		// Provide a helpful hint if the Archive folder doesn't exist.
		if strings.Contains(err.Error(), "TRYCREATE") || strings.Contains(err.Error(), "Mailbox doesn't exist") {
			return nil, &connectors.ExternalError{
				Message: fmt.Sprintf("Archive folder not found on server — the mailbox %q may not exist. Ensure Proton Mail Bridge is configured correctly: %v", archiveMailbox, err),
			}
		}
		return nil, imapErr
	}

	return connectors.JSONResult(map[string]any{
		"status":      "archived",
		"folder":      params.Folder,
		"archived":    len(params.MessageIDs),
		"message_ids": params.MessageIDs,
	})
}
