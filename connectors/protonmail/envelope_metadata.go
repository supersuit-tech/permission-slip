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
