package protonmail

import (
	"context"
	"fmt"
	"strings"

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

type archiveEmailParams = uidMessageParams

func parseArchiveParams(raw []byte) (*archiveEmailParams, error) {
	return parseUIDMessageParams(raw)
}

func (p *archiveEmailParams) validate() error {
	if err := (*uidMessageParams)(p).validate(); err != nil {
		return err
	}
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

	moveCmd := session.client.Move(uidSetFromMessageIDs(params.MessageIDs), archiveMailbox)
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
