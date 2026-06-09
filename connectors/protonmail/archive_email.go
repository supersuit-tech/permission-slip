package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// archiveMailbox is the IMAP folder that both Proton Mail Bridge and hydroxide
// expose for Proton's Archive system label (LabelArchive in hydroxide's source).
const archiveMailbox = "Archive"

// archiveEmailAction moves one or more emails to the Archive folder via IMAP
// UID MOVE (RFC 6851). Both Bridge and hydroxide support MOVE.
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

	if len(r.MessageIDs) > 0 && string(r.MessageIDs) != "null" {
		if err := json.Unmarshal(r.MessageIDs, &params.MessageIDs); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid message_ids: %v", err)}
		}
	}

	if r.MessageID != nil {
		params.MessageIDs = append(params.MessageIDs, *r.MessageID)
	}

	return params, nil
}

func (p *archiveEmailParams) validate() error {
	if len(p.MessageIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: provide message_id (single) or message_ids (array)"}
	}

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

	mboxData, err := session.selectMailboxReadWrite(params.Folder)
	if err != nil {
		return nil, err
	}
	if err := syncUIDValidity(params.Folder, mboxData, req.MailboxUIDValidity, uidValidityVerify); err != nil {
		return nil, err
	}

	var uidSet imap.UIDSet
	for _, id := range params.MessageIDs {
		uidSet.AddNum(imap.UID(id))
	}

	moveCmd := session.client.Move(uidSet, archiveMailbox)
	if _, err := moveCmd.Wait(); err != nil {
		imapErr := mapIMAPError(err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "TRYCREATE") || strings.Contains(errMsg, "Mailbox doesn't exist") {
			return nil, &connectors.ExternalError{
				Message: fmt.Sprintf("Archive folder not found on server — the mailbox %q may not exist. Ensure your local Proton IMAP/SMTP proxy (Bridge or hydroxide) is configured correctly and exposes an Archive folder: %v", archiveMailbox, err),
			}
		}
		if strings.Contains(strings.ToUpper(errMsg), "UID") || strings.Contains(strings.ToLower(errMsg), "not found") {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("one or more message UIDs not found in folder %q", params.Folder),
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
