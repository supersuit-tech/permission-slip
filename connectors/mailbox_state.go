package connectors

// MailboxUIDValidityStore tracks the last known IMAP UIDVALIDITY per mailbox
// folder for a credential. Implementations persist across requests so agents
// can pass stable {folder, uid} handles without carrying UIDVALIDITY.
type MailboxUIDValidityStore interface {
	UIDValidity(folder string) (validity uint32, known bool)
	SetUIDValidity(folder string, validity uint32) error
}
