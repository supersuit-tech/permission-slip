package protonmail

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type uidValidityMode int

const (
	uidValidityRecord uidValidityMode = iota
	uidValidityVerify
)

// syncUIDValidity records or verifies UIDVALIDITY for a folder.
//
// List operations use record mode (always update the stored value). Execute
// operations use verify mode: refuse when the stored value exists and differs
// from the server, otherwise record on first sight.
func syncUIDValidity(folder string, mboxData *imap.SelectData, store connectors.MailboxUIDValidityStore, mode uidValidityMode) error {
	if store == nil || mboxData == nil {
		return nil
	}

	current := mboxData.UIDValidity
	switch mode {
	case uidValidityRecord:
		return store.SetUIDValidity(folder, current)
	case uidValidityVerify:
		stored, known := store.UIDValidity(folder)
		if known && stored != current {
			return &connectors.ValidationError{
				Message: fmt.Sprintf(
					"mailbox %q state is no longer valid (UIDVALIDITY changed); re-list messages before acting on them",
					folder,
				),
			}
		}
		if !known {
			return store.SetUIDValidity(folder, current)
		}
		return nil
	default:
		return nil
	}
}
