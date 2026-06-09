package protonmail

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestResolveResourceDetails_ReadEmail_PopulatesMetadata(t *testing.T) {
	t.Parallel()

	orig := resolveMessageEnvelopes
	t.Cleanup(func() { resolveMessageEnvelopes = orig })

	resolveMessageEnvelopes = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, folder string, uids []uint32, _ connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		if folder != "INBOX" {
			t.Fatalf("folder = %q, want INBOX", folder)
		}
		if len(uids) != 1 || uids[0] != 10 {
			t.Fatalf("uids = %v, want [10]", uids)
		}
		return map[uint32]emailEnvelopeMetadata{
			10: {
				Subject: "Weekly Update",
				From:    []string{"alice@example.com"},
				To:      []string{"bob@example.com"},
				Date:    "2026-01-15T10:00:00Z",
			},
		}, nil
	}

	conn := New()
	params, _ := json.Marshal(map[string]any{"message_id": 10, "folder": "INBOX"})
	details, err := conn.ResolveResourceDetails(context.Background(), "protonmail.read_email", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertEnvelopeMetadataOnly(t, details)
	if details["subject"] != "Weekly Update" {
		t.Errorf("subject = %v", details["subject"])
	}
	if !reflect.DeepEqual(details["from"], []any{"alice@example.com"}) {
		t.Errorf("from = %v", details["from"])
	}
	if !reflect.DeepEqual(details["to"], []any{"bob@example.com"}) {
		t.Errorf("to = %v", details["to"])
	}
	if details["date"] != "2026-01-15T10:00:00Z" {
		t.Errorf("date = %v", details["date"])
	}
}

func TestResolveResourceDetails_ReadEmail_FailureFallsBack(t *testing.T) {
	t.Parallel()

	orig := resolveMessageEnvelopes
	t.Cleanup(func() { resolveMessageEnvelopes = orig })

	resolveMessageEnvelopes = func(context.Context, *ProtonMailConnector, connectors.Credentials, string, []uint32, connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		return nil, &connectors.ExternalError{Message: "proxy down"}
	}

	conn := New()
	params, _ := json.Marshal(map[string]any{"message_id": 10})
	details, err := conn.ResolveResourceDetails(context.Background(), "protonmail.read_email", params, validCreds())
	if err != nil {
		t.Fatalf("expected nil error for best-effort fallback, got %v", err)
	}
	if details != nil {
		t.Fatalf("expected nil details on failure, got %v", details)
	}
}

func TestResolveResourceDetails_ArchiveEmail_SingleUsesFlatFields(t *testing.T) {
	t.Parallel()

	orig := resolveMessageEnvelopes
	t.Cleanup(func() { resolveMessageEnvelopes = orig })

	resolveMessageEnvelopes = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, uids []uint32, _ connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		return map[uint32]emailEnvelopeMetadata{
			uids[0]: {Subject: "Archive me", From: []string{"sender@example.com"}, To: []string{"me@example.com"}, Date: "2026-02-01T12:00:00Z"},
		}, nil
	}

	conn := New()
	params, _ := json.Marshal(map[string]any{"message_id": 42})
	details, err := conn.ResolveResourceDetails(context.Background(), "protonmail.archive_email", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := details["messages"]; ok {
		t.Fatalf("single archive should not use messages map, got %v", details)
	}
	assertEnvelopeMetadataOnly(t, details)
	if details["subject"] != "Archive me" {
		t.Errorf("subject = %v", details["subject"])
	}
}

func TestResolveResourceDetails_ArchiveEmail_BatchKeyedByHandle(t *testing.T) {
	t.Parallel()

	orig := resolveMessageEnvelopes
	t.Cleanup(func() { resolveMessageEnvelopes = orig })

	resolveMessageEnvelopes = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, uids []uint32, _ connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		out := make(map[uint32]emailEnvelopeMetadata, len(uids))
		for _, uid := range uids {
			out[uid] = emailEnvelopeMetadata{
				Subject: "Subject",
				From:    []string{"from@example.com"},
				To:      []string{"to@example.com"},
				Date:    "2026-03-01T09:00:00Z",
			}
		}
		return out, nil
	}

	conn := New()
	params, _ := json.Marshal(map[string]any{"message_ids": []int{10, 11}})
	details, err := conn.ResolveResourceDetails(context.Background(), "protonmail.archive_email", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rawMessages, ok := details["messages"].(map[string]any)
	if !ok {
		t.Fatalf("expected messages map, got %T", details["messages"])
	}
	if len(rawMessages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(rawMessages))
	}
	for _, key := range []string{"10", "11"} {
		meta, ok := rawMessages[key].(map[string]any)
		if !ok {
			t.Fatalf("messages[%q] type = %T", key, rawMessages[key])
		}
		assertEnvelopeMetadataOnly(t, meta)
	}
}

func TestResolveResourceDetails_PassesMailboxStoreFromContext(t *testing.T) {
	t.Parallel()

	orig := resolveMessageEnvelopes
	t.Cleanup(func() { resolveMessageEnvelopes = orig })

	store := &stubUIDValidityStore{validities: map[string]uint32{"INBOX": 1}}
	var gotStore connectors.MailboxUIDValidityStore
	resolveMessageEnvelopes = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ string, _ []uint32, mailboxStore connectors.MailboxUIDValidityStore) (map[uint32]emailEnvelopeMetadata, error) {
		gotStore = mailboxStore
		return nil, nil
	}

	conn := New()
	params, _ := json.Marshal(map[string]any{"message_id": 10})
	ctx := connectors.ContextWithMailboxUIDValidity(context.Background(), store)
	_, err := conn.ResolveResourceDetails(ctx, "protonmail.read_email", params, validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotStore != store {
		t.Fatal("expected mailbox UID validity store from context")
	}
}

func TestEnvelopeToMetadata_OnlyAllowedFields(t *testing.T) {
	t.Parallel()

	meta := envelopeToMetadata(nil)
	if meta.Subject != "" || meta.Date != "" || len(meta.From) != 0 || len(meta.To) != 0 {
		t.Fatalf("empty envelope should yield zero metadata: %+v", meta)
	}
	m := meta.asMap()
	if len(m) != 4 {
		t.Fatalf("expected exactly 4 keys, got %d: %v", len(m), m)
	}
	for _, key := range []string{"subject", "from", "to", "date"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	for _, forbidden := range []string{"body", "attachments", "flags", "uid"} {
		if _, ok := m[forbidden]; ok {
			t.Errorf("forbidden key %q present", forbidden)
		}
	}
}

func assertEnvelopeMetadataOnly(t *testing.T, details map[string]any) {
	t.Helper()
	if len(details) != 4 {
		t.Fatalf("expected exactly 4 metadata fields, got %d: %v", len(details), details)
	}
	for _, key := range []string{"subject", "from", "to", "date"} {
		if _, ok := details[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	for _, forbidden := range []string{"body", "attachments", "flags", "uid"} {
		if _, ok := details[forbidden]; ok {
			t.Errorf("forbidden key %q leaked into resource_details", forbidden)
		}
	}
}

type stubUIDValidityStore struct {
	validities map[string]uint32
}

func (s *stubUIDValidityStore) UIDValidity(folder string) (uint32, bool) {
	v, ok := s.validities[folder]
	return v, ok
}

func (s *stubUIDValidityStore) SetUIDValidity(folder string, validity uint32) error {
	if s.validities == nil {
		s.validities = make(map[string]uint32)
	}
	s.validities[folder] = validity
	return nil
}
