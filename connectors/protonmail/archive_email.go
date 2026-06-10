package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

// archiveExpandUIDsFn expands archive targets to full conversations. Tests may
// replace it to avoid a live proxy.
var archiveExpandUIDsFn = expandArchiveUIDs

// archivePerformMove executes the UID MOVE to the Archive folder. Tests may
// replace it to avoid a live proxy.
var archivePerformMove = func(session *imapSession, uids []uint32, dest string) error {
	moveCmd := session.client.Move(uidSetFromMessageIDs(uids), dest)
	_, err := moveCmd.Wait()
	return err
}

// archiveConnectAndSelect opens a read-write IMAP session and verifies
// UIDVALIDITY. Tests may replace it to avoid a live proxy.
var archiveConnectAndSelect = func(creds connectors.Credentials, timeout time.Duration, folder string, store connectors.MailboxUIDValidityStore) (*imapSession, error) {
	session, err := connectIMAP(creds, timeout)
	if err != nil {
		return nil, err
	}
	mboxData, err := session.selectMailboxReadWrite(folder)
	if err != nil {
		session.close()
		return nil, err
	}
	if err := syncUIDValidity(folder, mboxData, store, uidValidityVerify); err != nil {
		session.close()
		return nil, err
	}
	return session, nil
}

type archiveEmailParams struct {
	MessageIDs    []uint32 `json:"-"`
	Folder        string   `json:"folder"`
	IncludeThread *bool    `json:"include_thread"`
}

func parseArchiveParams(raw []byte) (*archiveEmailParams, error) {
	var r struct {
		uidMessageRaw
		IncludeThread *bool `json:"include_thread"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	base := &uidMessageParams{Folder: r.Folder}
	if len(r.MessageIDs) > 0 && string(r.MessageIDs) != "null" {
		if err := json.Unmarshal(r.MessageIDs, &base.MessageIDs); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid message_ids: %v", err)}
		}
	}
	if r.MessageID != nil {
		base.MessageIDs = append(base.MessageIDs, *r.MessageID)
	}

	return &archiveEmailParams{
		MessageIDs:    base.MessageIDs,
		Folder:        base.Folder,
		IncludeThread: r.IncludeThread,
	}, nil
}

func validateArchiveParams(p *archiveEmailParams) error {
	base := &uidMessageParams{MessageIDs: p.MessageIDs, Folder: p.Folder}
	if err := base.validate(); err != nil {
		return err
	}
	p.MessageIDs = base.MessageIDs
	p.Folder = base.Folder
	if strings.EqualFold(p.Folder, archiveMailbox) {
		return &connectors.ValidationError{Message: "cannot archive emails that are already in the Archive folder"}
	}
	return nil
}

func (a *archiveEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseArchiveParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := validateArchiveParams(params); err != nil {
		return nil, err
	}

	session, err := archiveConnectAndSelect(req.Credentials, a.conn.timeout, params.Folder, req.MailboxUIDValidity)
	if err != nil {
		return nil, err
	}
	defer session.close()

	uidsToArchive := params.MessageIDs
	var threadExpanded bool
	if includeThreadEnabled(params.IncludeThread) {
		expanded, err := archiveExpandUIDsFn(session, params.MessageIDs)
		if err != nil {
			return nil, err
		}
		uidsToArchive = expanded
		threadExpanded = !sameUIDSet(expanded, params.MessageIDs)
	}

	if err := archivePerformMove(session, uidsToArchive, archiveMailbox); err != nil {
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

	result := map[string]any{
		"status":      "archived",
		"folder":      params.Folder,
		"archived":    len(uidsToArchive),
		"message_ids": params.MessageIDs,
	}
	if includeThreadEnabled(params.IncludeThread) && threadExpanded {
		result["thread_expanded"] = true
		result["archived_uids"] = uidsToArchive
	}
	return connectors.JSONResult(result)
}

func sameUIDSet(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[uint32]struct{}, len(a))
	for _, uid := range a {
		seen[uid] = struct{}{}
	}
	for _, uid := range b {
		if _, ok := seen[uid]; !ok {
			return false
		}
	}
	return true
}
