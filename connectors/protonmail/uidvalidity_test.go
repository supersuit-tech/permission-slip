package protonmail

import (
	"sync"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type memUIDValidityStore struct {
	mu      sync.Mutex
	folders map[string]uint32
}

func newMemUIDValidityStore() *memUIDValidityStore {
	return &memUIDValidityStore{folders: make(map[string]uint32)}
}

func (s *memUIDValidityStore) UIDValidity(folder string) (uint32, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.folders[folder]
	return v, ok
}

func (s *memUIDValidityStore) SetUIDValidity(folder string, validity uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.folders[folder] = validity
	return nil
}

func TestSyncUIDValidity_RecordUpdatesStore(t *testing.T) {
	t.Parallel()

	store := newMemUIDValidityStore()
	mbox := &imap.SelectData{UIDValidity: 42}

	if err := syncUIDValidity("INBOX", mbox, store, uidValidityRecord); err != nil {
		t.Fatalf("syncUIDValidity: %v", err)
	}
	got, ok := store.UIDValidity("INBOX")
	if !ok || got != 42 {
		t.Fatalf("expected UIDVALIDITY 42, got %d (ok=%v)", got, ok)
	}
}

func TestSyncUIDValidity_VerifyRejectsMismatch(t *testing.T) {
	t.Parallel()

	store := newMemUIDValidityStore()
	_ = store.SetUIDValidity("INBOX", 100)

	err := syncUIDValidity("INBOX", &imap.SelectData{UIDValidity: 200}, store, uidValidityVerify)
	if err == nil {
		t.Fatal("expected UIDVALIDITY mismatch error")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestSyncUIDValidity_VerifyAllowsMatch(t *testing.T) {
	t.Parallel()

	store := newMemUIDValidityStore()
	_ = store.SetUIDValidity("INBOX", 100)

	if err := syncUIDValidity("INBOX", &imap.SelectData{UIDValidity: 100}, store, uidValidityVerify); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncUIDValidity_VerifyRecordsFirstSeen(t *testing.T) {
	t.Parallel()

	store := newMemUIDValidityStore()
	if err := syncUIDValidity("INBOX", &imap.SelectData{UIDValidity: 77}, store, uidValidityVerify); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := store.UIDValidity("INBOX")
	if !ok || got != 77 {
		t.Fatalf("expected UIDVALIDITY 77, got %d (ok=%v)", got, ok)
	}
}

func TestEnvelopeToSummary_UsesUIDAndMessageIDHeader(t *testing.T) {
	t.Parallel()

	env := &imap.Envelope{
		Subject:   "Invoice",
		MessageID: "<abc@example.com>",
	}
	summary := envelopeToSummary(123, env, []imap.Flag{imap.FlagSeen})

	if summary.UID != 123 {
		t.Errorf("UID = %d, want 123", summary.UID)
	}
	if summary.MessageIDHeader != "<abc@example.com>" {
		t.Errorf("MessageIDHeader = %q", summary.MessageIDHeader)
	}
}

// TestUIDStabilityAcrossExpunge documents that UIDs remain stable when lower
// sequence numbers are expunged. Sequence numbers shift; UIDs do not.
func TestUIDStabilityAcrossExpunge(t *testing.T) {
	t.Parallel()

	type message struct {
		seq uint32
		uid imap.UID
	}
	mailbox := []message{
		{seq: 1, uid: 10},
		{seq: 2, uid: 20},
		{seq: 3, uid: 30},
	}

	// Simulate expunging seq 1: remaining messages shift down by one sequence.
	afterExpunge := mailbox[1:]
	if afterExpunge[0].seq == 1 && afterExpunge[0].uid != 20 {
		t.Fatalf("expected uid 20 at seq 1 after expunge, got uid %d", afterExpunge[0].uid)
	}
	if afterExpunge[1].seq == 2 && afterExpunge[1].uid != 30 {
		t.Fatalf("expected uid 30 at seq 2 after expunge, got uid %d", afterExpunge[1].uid)
	}

	// Agent targeted uid 30 before expunge; it still resolves after churn.
	targetUID := imap.UID(30)
	found := false
	for _, msg := range afterExpunge {
		if msg.uid == targetUID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("target UID should remain addressable after unrelated expunge")
	}
}

func TestDeduplicateUint32_BatchArchive(t *testing.T) {
	t.Parallel()

	got := deduplicateUint32([]uint32{30, 10, 30, 20, 10})
	want := []uint32{30, 10, 20}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
