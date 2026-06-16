package protonmail

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type removeLabelAction struct {
	conn *ProtonMailConnector
}

// removeLabelExpandUIDsFn expands remove targets to full conversations. Tests may
// replace it to avoid a live proxy.
var removeLabelExpandUIDsFn = expandArchiveUIDs

// removeLabelFetchEnvelopesFn fetches source-folder envelopes for label removal.
// Tests may replace it to avoid a live proxy.
var removeLabelFetchEnvelopesFn = func(session *imapSession, uidSet imap.UIDSet) ([]emailSummary, error) {
	return session.fetchEnvelopes(uidSet)
}

// removeLabelFindLabelUIDsFn maps source messages to UIDs in the label mailbox.
// Tests may replace it to avoid a live proxy.
var removeLabelFindLabelUIDsFn = findLabelMailboxUIDs

// removeLabelSelectLabelMailbox selects the label mailbox read-write. Tests may
// replace it to avoid a live proxy.
var removeLabelSelectLabelMailbox = func(session *imapSession, labelMailbox string) error {
	_, err := session.selectMailboxReadWrite(labelMailbox)
	if err != nil {
		return mapLabelMailboxError(err, labelMailbox)
	}
	return nil
}

// removeLabelMarkDeletedAndExpunge marks label-mailbox copies deleted and
// expunges them. Tests may replace it to avoid a live proxy.
var removeLabelMarkDeletedAndExpunge = func(session *imapSession, labelUIDs []uint32) error {
	uidSet := uidSetFromMessageIDs(labelUIDs)
	if err := storeMessageFlags(session, uidSet, imap.StoreFlagsAdd, []imap.Flag{imap.FlagDeleted}); err != nil {
		return mapIMAPError(err)
	}
	expungeCmd := session.client.UIDExpunge(uidSet)
	if _, err := expungeCmd.Collect(); err != nil {
		return mapIMAPError(err)
	}
	return nil
}

// removeLabelConnectAndSelect opens a read-write IMAP session in the source
// folder and verifies UIDVALIDITY. Tests may replace it to avoid a live proxy.
var removeLabelConnectAndSelect = func(creds connectors.Credentials, timeout time.Duration, folder string, store connectors.MailboxUIDValidityStore) (*imapSession, error) {
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

func (a *removeLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseLabelMessageParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	session, err := removeLabelConnectAndSelect(req.Credentials, a.conn.timeout, params.Folder, req.MailboxUIDValidity)
	if err != nil {
		return nil, err
	}
	defer session.close()

	uidsToProcess := params.MessageIDs
	var threadExpanded bool
	if includeThreadEnabled(params.IncludeThread) {
		expanded, err := removeLabelExpandUIDsFn(session, params.MessageIDs)
		if err != nil {
			return nil, err
		}
		uidsToProcess = expanded
		threadExpanded = !sameUIDSet(expanded, params.MessageIDs)
	}

	summaries, err := removeLabelFetchEnvelopesFn(session, uidSetFromMessageIDs(uidsToProcess))
	if err != nil {
		return nil, mapUIDNotFoundError(err, params.Folder)
	}
	if len(summaries) == 0 {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("one or more message UIDs not found in folder %q", params.Folder),
		}
	}

	if err := removeLabelSelectLabelMailbox(session, params.LabelMailbox); err != nil {
		return nil, err
	}

	labelUIDs, err := removeLabelFindLabelUIDsFn(session, summaries)
	if err != nil {
		return nil, err
	}

	if err := removeLabelMarkDeletedAndExpunge(session, labelUIDs); err != nil {
		return nil, err
	}

	result := map[string]any{
		"status":        "label_removed",
		"folder":        params.Folder,
		"label":         labelDisplayName(params.LabelMailbox),
		"label_mailbox": params.LabelMailbox,
		"removed":       len(labelUIDs),
		"message_ids":   params.MessageIDs,
	}
	if includeThreadEnabled(params.IncludeThread) && threadExpanded {
		result["thread_expanded"] = true
		result["removed_uids"] = uidsToProcess
	}
	return connectors.JSONResult(result)
}
