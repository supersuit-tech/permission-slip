package connectors

import "context"

// MailboxUIDValidityStore tracks the last known IMAP UIDVALIDITY per mailbox
// folder for a credential. Implementations persist across requests so agents
// can pass stable {folder, uid} handles without carrying UIDVALIDITY.
type MailboxUIDValidityStore interface {
	UIDValidity(folder string) (validity uint32, known bool)
	SetUIDValidity(folder string, validity uint32) error
}

type mailboxUIDValidityContextKey struct{}

// ContextWithMailboxUIDValidity attaches a UIDVALIDITY store for connectors that
// need mailbox state during best-effort resource detail resolution.
func ContextWithMailboxUIDValidity(ctx context.Context, store MailboxUIDValidityStore) context.Context {
	if store == nil {
		return ctx
	}
	return context.WithValue(ctx, mailboxUIDValidityContextKey{}, store)
}

// MailboxUIDValidityFromContext returns the store attached by
// ContextWithMailboxUIDValidity, or nil when absent.
func MailboxUIDValidityFromContext(ctx context.Context) MailboxUIDValidityStore {
	store, _ := ctx.Value(mailboxUIDValidityContextKey{}).(MailboxUIDValidityStore)
	return store
}
