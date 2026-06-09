package protonmail

import (
	"context"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// executeUIDMessageAction runs a read-write mailbox operation that requires
// UIDVALIDITY verification and a UID set derived from message parameters.
func executeUIDMessageAction(
	ctx context.Context,
	conn *ProtonMailConnector,
	req connectors.ActionRequest,
	params *uidMessageParams,
	run func(session *imapSession, uidSet imap.UIDSet) error,
) error {
	session, err := connectIMAP(req.Credentials, conn.timeout)
	if err != nil {
		return err
	}
	defer session.close()

	mboxData, err := session.selectMailboxReadWrite(params.Folder)
	if err != nil {
		return err
	}
	if err := syncUIDValidity(params.Folder, mboxData, req.MailboxUIDValidity, uidValidityVerify); err != nil {
		return err
	}

	return run(session, uidSetFromMessageIDs(params.MessageIDs))
}

func mapUIDNotFoundError(err error, folder string) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	if strings.Contains(strings.ToUpper(errMsg), "UID") || strings.Contains(strings.ToLower(errMsg), "not found") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("one or more message UIDs not found in folder %q", folder),
		}
	}
	return mapIMAPError(err)
}

func storeMessageFlags(session *imapSession, uidSet imap.UIDSet, op imap.StoreFlagsOp, flags []imap.Flag) error {
	storeCmd := session.client.Store(uidSet, &imap.StoreFlags{
		Op:     op,
		Flags:  flags,
		Silent: true,
	}, nil)
	return storeCmd.Close()
}
