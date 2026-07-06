package protonmail

import (
	"context"
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestResolveMessageEnvelopesWithRetry_SucceedsOnSecondAttempt(t *testing.T) {
	orig := resolveMessageEnvelopesIMAP
	t.Cleanup(func() { resolveMessageEnvelopesIMAP = orig })

	attempts := 0
	resolveMessageEnvelopesIMAP = func(context.Context, *ProtonMailConnector, connectors.Credentials, string, []uint32, connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("temporary imap failure")
		}
		return map[uint32]emailEnvelopeMetadata{
			42: {Subject: "Hello", From: []string{"sender@example.com"}},
		}, nil
	}

	meta, err := resolveMessageEnvelopesWithRetry(context.Background(), &ProtonMailConnector{}, connectors.NewCredentials(nil), "INBOX", []uint32{42}, nil)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if meta[42].Subject != "Hello" {
		t.Fatalf("meta = %#v", meta[42])
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}
