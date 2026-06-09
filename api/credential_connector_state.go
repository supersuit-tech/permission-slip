package api

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

// protonmailUIDValidityStore persists IMAP UIDVALIDITY per folder on a credential.
type protonmailUIDValidityStore struct {
	ctx          context.Context
	deps         *Deps
	credentialID string
}

func (s *protonmailUIDValidityStore) UIDValidity(folder string) (uint32, bool) {
	validity, known, err := db.GetProtonmailUIDValidity(s.ctx, s.deps.DB, s.credentialID, folder)
	if err != nil {
		return 0, false
	}
	return validity, known
}

func (s *protonmailUIDValidityStore) SetUIDValidity(folder string, validity uint32) error {
	return db.SetProtonmailUIDValidity(s.ctx, s.deps.DB, s.credentialID, folder, validity)
}

func newProtonmailUIDValidityStore(ctx context.Context, deps *Deps, credentialID string) connectors.MailboxUIDValidityStore {
	if credentialID == "" {
		return nil
	}
	return &protonmailUIDValidityStore{
		ctx:          ctx,
		deps:         deps,
		credentialID: credentialID,
	}
}
