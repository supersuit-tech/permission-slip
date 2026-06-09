package protonmail

import (
	"time"

	"github.com/emersion/go-imap/v2"
)

// emailEnvelopeMetadata is the approval-safe subset of email metadata exposed
// via resource_details. Never include body, attachments, or flags.
type emailEnvelopeMetadata struct {
	Subject string   `json:"subject"`
	From    []string `json:"from"`
	To      []string `json:"to"`
	Date    string   `json:"date"`
}

// emailEnvelopeWithMessageID adds the RFC Message-ID header for reply threading.
// MessageIDHeader is never stored in approval resource_details.
type emailEnvelopeWithMessageID struct {
	emailEnvelopeMetadata
	MessageIDHeader string
}

func envelopeToMetadata(env *imap.Envelope) emailEnvelopeMetadata {
	if env == nil {
		return emailEnvelopeMetadata{}
	}
	return emailEnvelopeMetadata{
		Subject: env.Subject,
		Date:    env.Date.Format(time.RFC3339),
		From:    formatAddresses(env.From),
		To:      formatAddresses(env.To),
	}
}

func (m emailEnvelopeMetadata) asMap() map[string]any {
	return map[string]any{
		"subject": m.Subject,
		"from":    m.From,
		"to":      m.To,
		"date":    m.Date,
	}
}

// fetchEnvelopeMetadataByUID performs a metadata-only UID FETCH (envelope only;
// no body, attachments, or flags) so messages are not marked read.
func fetchEnvelopeMetadataByUID(session *imapSession, uidSet imap.UIDSet) (map[uint32]emailEnvelopeMetadata, error) {
	if len(uidSet) == 0 {
		return nil, nil
	}

	fetchCmd := session.client.Fetch(uidSet, &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	})
	defer fetchCmd.Close()

	out := make(map[uint32]emailEnvelopeMetadata)
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		buf, err := msg.Collect()
		if err != nil {
			return nil, mapIMAPError(err)
		}
		if buf.UID == 0 || buf.Envelope == nil {
			continue
		}
		out[uint32(buf.UID)] = envelopeToMetadata(buf.Envelope)
	}
	return out, nil
}

func resolveMessageEnvelopesWithMessageID(ctx context.Context, conn *ProtonMailConnector, creds connectors.Credentials, folder string, uids []uint32, store connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeWithMessageID, error) {
	if len(uids) == 0 {
		return nil, nil
	}
	if username, ok := creds.Get(credKeyUsername); !ok || username == "" {
		return nil, nil
	}
	if password, ok := creds.Get(credKeyPassword); !ok || password == "" {
		return nil, nil
	}

	timeout := conn.timeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	session, err := connectIMAP(creds, timeout)
	if err != nil {
		return nil, err
	}
	defer session.close()

	mboxData, err := session.selectMailbox(folder)
	if err != nil {
		return nil, err
	}
	if err := syncUIDValidity(folder, mboxData, store, uidValidityVerify); err != nil {
		return nil, err
	}

	var uidSet imap.UIDSet
	for _, id := range uids {
		uidSet.AddNum(imap.UID(id))
	}

	return fetchEnvelopeWithMessageIDByUID(session, uidSet)
}

func fetchEnvelopeWithMessageIDByUID(session *imapSession, uidSet imap.UIDSet) (map[uint32]emailEnvelopeWithMessageID, error) {
	if len(uidSet) == 0 {
		return nil, nil
	}

	fetchCmd := session.client.Fetch(uidSet, &imap.FetchOptions{
		Envelope: true,
		UID:      true,
	})
	defer fetchCmd.Close()

	out := make(map[uint32]emailEnvelopeWithMessageID)
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		buf, err := msg.Collect()
		if err != nil {
			return nil, mapIMAPError(err)
		}
		if buf.UID == 0 || buf.Envelope == nil {
			continue
		}
		meta := envelopeToMetadata(buf.Envelope)
		out[uint32(buf.UID)] = emailEnvelopeWithMessageID{
			emailEnvelopeMetadata: meta,
			MessageIDHeader:       buf.Envelope.MessageID,
		}
	}
	return out, nil
}
