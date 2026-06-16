package protonmail

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type applyLabelAction struct {
	conn *ProtonMailConnector
}

// applyLabelExpandUIDsFn expands label targets to full conversations. Tests may
// replace it to avoid a live proxy.
var applyLabelExpandUIDsFn = expandArchiveUIDs

// applyLabelPerformCopy executes the UID COPY into the label mailbox. Tests may
// replace it to avoid a live proxy.
var applyLabelPerformCopy = func(session *imapSession, uids []uint32, labelMailbox string) error {
	copyCmd := session.client.Copy(uidSetFromMessageIDs(uids), labelMailbox)
	_, err := copyCmd.Wait()
	return err
}

// applyLabelConnectAndSelect opens a read-write IMAP session in the source
// folder and verifies UIDVALIDITY. Tests may replace it to avoid a live proxy.
var applyLabelConnectAndSelect = func(creds connectors.Credentials, timeout time.Duration, folder string, store connectors.MailboxUIDValidityStore) (*imapSession, error) {
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

func (a *applyLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseLabelMessageParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	session, err := applyLabelConnectAndSelect(req.Credentials, a.conn.timeout, params.Folder, req.MailboxUIDValidity)
	if err != nil {
		return nil, err
	}
	defer session.close()

	uidsToLabel := params.MessageIDs
	var threadExpanded bool
	if includeThreadEnabled(params.IncludeThread) {
		expanded, err := applyLabelExpandUIDsFn(session, params.MessageIDs)
		if err != nil {
			return nil, err
		}
		uidsToLabel = expanded
		threadExpanded = !sameUIDSet(expanded, params.MessageIDs)
	}

	if err := applyLabelPerformCopy(session, uidsToLabel, params.LabelMailbox); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "TRYCREATE") || strings.Contains(errMsg, "Mailbox doesn't exist") {
			return nil, &connectors.ExternalError{
				Message: fmt.Sprintf("label mailbox %q not found on server — ensure the label exists (call protonmail.list_labels to discover valid names): %v", params.LabelMailbox, err),
			}
		}
		if strings.Contains(strings.ToUpper(errMsg), "UID") || strings.Contains(strings.ToLower(errMsg), "not found") {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("one or more message UIDs not found in folder %q", params.Folder),
			}
		}
		return nil, mapIMAPError(err)
	}

	result := map[string]any{
		"status":        "label_applied",
		"folder":        params.Folder,
		"label":         labelDisplayName(params.LabelMailbox),
		"label_mailbox": params.LabelMailbox,
		"labeled":       len(uidsToLabel),
		"message_ids":   params.MessageIDs,
	}
	if includeThreadEnabled(params.IncludeThread) && threadExpanded {
		result["thread_expanded"] = true
		result["labeled_uids"] = uidsToLabel
	}
	return connectors.JSONResult(result)
}
