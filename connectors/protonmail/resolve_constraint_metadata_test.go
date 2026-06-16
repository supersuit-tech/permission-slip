package protonmail

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestPrimarySenderAddress(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   []string
		want string
		ok   bool
	}{
		{[]string{"alice@example.com"}, "alice@example.com", true},
		{[]string{"Alice <alice@example.com>"}, "alice@example.com", true},
		{[]string{}, "", false},
		{[]string{""}, "", false},
	}
	for _, tc := range tests {
		got, ok := primarySenderAddress(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("primarySenderAddress(%v) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestResolveArchiveConstraintMetadata_SingleMessage(t *testing.T) {
	t.Parallel()
	orig := resolveMessageEnvelopes
	t.Cleanup(func() { resolveMessageEnvelopes = orig })

	resolveMessageEnvelopes = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, folder string, uids []uint32, _ connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		if folder != "INBOX" || len(uids) != 1 || uids[0] != 99 {
			t.Fatalf("unexpected fetch: folder=%q uids=%v", folder, uids)
		}
		return map[uint32]emailEnvelopeMetadata{
			99: {From: []string{"Alice <alice@example.com>"}},
		}, nil
	}

	conn := New()
	params := json.RawMessage(`{"message_id":99,"folder":"INBOX","include_thread":false}`)
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "protonmail.archive_email", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	if meta["sender"] != "alice@example.com" {
		t.Fatalf("sender = %v, want alice@example.com", meta["sender"])
	}
}

func TestResolveArchiveConstraintMetadata_ThreadExpansionUsesAllSenders(t *testing.T) {
	t.Parallel()
	origEnvelopes := resolveMessageEnvelopes
	origExpand := expandArchiveUIDsForApproval
	t.Cleanup(func() {
		resolveMessageEnvelopes = origEnvelopes
		expandArchiveUIDsForApproval = origExpand
	})

	expandArchiveUIDsForApproval = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ []uint32) ([]uint32, error) {
		return []uint32{10, 11}, nil
	}
	resolveMessageEnvelopes = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, uids []uint32, _ connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		return map[uint32]emailEnvelopeMetadata{
			10: {From: []string{"alice@example.com"}},
			11: {From: []string{"bob@example.com"}},
		}, nil
	}

	conn := New()
	params := json.RawMessage(`{"message_id":10,"folder":"INBOX","include_thread":true}`)
	meta, err := conn.ResolveConstraintMetadata(context.Background(), "protonmail.archive_email", params, validCreds())
	if err != nil {
		t.Fatalf("ResolveConstraintMetadata: %v", err)
	}
	senders, ok := meta["senders"].([]string)
	if !ok || len(senders) != 2 {
		t.Fatalf("senders = %v, want two addresses", meta["senders"])
	}
	if _, hasSingle := meta["sender"]; hasSingle {
		t.Fatal("expected no singular sender field for multi-message batch")
	}
}

func TestResolveArchiveConstraintMetadata_UnknownAction(t *testing.T) {
	t.Parallel()
	conn := New()
	_, err := conn.ResolveConstraintMetadata(context.Background(), "protonmail.read_inbox", json.RawMessage(`{}`), validCreds())
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}
