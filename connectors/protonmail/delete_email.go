package protonmail

import (
	"context"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// trashMailbox is the IMAP folder Proton Mail Bridge and hydroxide expose for
// the Trash system label.
const trashMailbox = "Trash"

type deleteEmailAction struct {
	conn *ProtonMailConnector
}

func (a *deleteEmailAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseUIDMessageParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	if strings.EqualFold(params.Folder, trashMailbox) {
		return nil, &connectors.ValidationError{Message: "cannot delete emails that are already in the Trash folder"}
	}

	err = executeUIDMessageAction(ctx, a.conn, req, params, func(session *imapSession, uidSet imap.UIDSet) error {
		moveCmd := session.client.Move(uidSet, trashMailbox)
		if _, err := moveCmd.Wait(); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "TRYCREATE") || strings.Contains(errMsg, "Mailbox doesn't exist") {
				return &connectors.ExternalError{
					Message: fmt.Sprintf("Trash folder not found on server — the mailbox %q may not exist. Ensure your local Proton IMAP/SMTP proxy exposes a Trash folder: %v", trashMailbox, err),
				}
			}
			return mapUIDNotFoundError(err, params.Folder)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"status":      "deleted",
		"folder":      params.Folder,
		"deleted":     len(params.MessageIDs),
		"message_ids": params.MessageIDs,
	})
}
