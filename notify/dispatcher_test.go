package notify

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// stubSender records calls and optionally returns an error.
type stubSender struct {
	name    string
	err     error
	mu      sync.Mutex
	calls   []Approval
	blocked chan struct{} // if non-nil, Send blocks until closed
	done    chan struct{} // if non-nil, closed after Send completes
}

func (s *stubSender) Name() string { return s.name }

func (s *stubSender) Send(_ context.Context, a Approval, _ Recipient) error {
	if s.blocked != nil {
		<-s.blocked
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, a)
	if s.done != nil {
		close(s.done)
	}
	return s.err
}

func (s *stubSender) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// stubPrefs returns a fixed enabled/disabled map. Channels not in the map
// are treated as enabled (matching the DB default behaviour).
type stubPrefs struct {
	disabled map[string]bool // channel name → true means disabled
	err      error           // simulate preference lookup failure
}

func (p *stubPrefs) IsChannelEnabled(_ context.Context, _ string, channel string) (bool, error) {
	if p.err != nil {
		return false, p.err
	}
	if p.disabled[channel] {
		return false, nil
	}
	return true, nil
}

func TestDispatcher_NilSafe(t *testing.T) {
	t.Parallel()
	var d *Dispatcher
	// Must not panic.
	d.Dispatch(context.Background(), testApproval(), testRecipient())
	d.DispatchSync(context.Background(), testApproval(), testRecipient())
}

func TestDispatcher_NoSenders(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(nil, nil)
	// Must not panic.
	d.Dispatch(context.Background(), testApproval(), testRecipient())
	d.DispatchSync(context.Background(), testApproval(), testRecipient())
}

func TestDispatcher_SingleSender(t *testing.T) {
	t.Parallel()
	s := &stubSender{name: "email"}
	d := NewDispatcher([]Sender{s}, nil)

	d.DispatchSync(context.Background(), testApproval(), testRecipient())

	if s.callCount() != 1 {
		t.Fatalf("expected 1 call, got %d", s.callCount())
	}
}

func TestDispatcher_MultipleSenders(t *testing.T) {
	t.Parallel()
	s1 := &stubSender{name: "email"}
	s2 := &stubSender{name: "sms"}
	s3 := &stubSender{name: "web-push"}
	d := NewDispatcher([]Sender{s1, s2, s3}, nil)

	d.DispatchSync(context.Background(), testApproval(), testRecipient())

	for _, s := range []*stubSender{s1, s2, s3} {
		if s.callCount() != 1 {
			t.Errorf("sender %q: expected 1 call, got %d", s.name, s.callCount())
		}
	}
}

func TestDispatcher_PartialFailure(t *testing.T) {
	t.Parallel()
	s1 := &stubSender{name: "email"}
	s2 := &stubSender{name: "sms", err: errors.New("twilio timeout")}
	s3 := &stubSender{name: "web-push"}
	d := NewDispatcher([]Sender{s1, s2, s3}, nil)

	// DispatchSync should not panic; errors are logged per-channel.
	d.DispatchSync(context.Background(), testApproval(), testRecipient())

	// All senders should have been called regardless of s2's failure.
	if s1.callCount() != 1 {
		t.Errorf("email: expected 1 call, got %d", s1.callCount())
	}
	if s2.callCount() != 1 {
		t.Errorf("sms: expected 1 call, got %d", s2.callCount())
	}
	if s3.callCount() != 1 {
		t.Errorf("web-push: expected 1 call, got %d", s3.callCount())
	}
}

func TestDispatcher_ConcurrentSending(t *testing.T) {
	t.Parallel()
	// Verify senders execute concurrently, not sequentially.
	blocker := make(chan struct{})
	s1 := &stubSender{name: "email", blocked: blocker}
	s2 := &stubSender{name: "sms", blocked: blocker}
	d := NewDispatcher([]Sender{s1, s2}, nil)

	done := make(chan struct{})
	go func() {
		d.DispatchSync(context.Background(), testApproval(), testRecipient())
		close(done)
	}()

	// Unblock both senders simultaneously — if dispatch were sequential,
	// only one would be waiting on the channel.
	close(blocker)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch timed out — senders may not be concurrent")
	}

	if s1.callCount() != 1 || s2.callCount() != 1 {
		t.Fatalf("expected 1 call each, got email=%d sms=%d", s1.callCount(), s2.callCount())
	}
}

func TestDispatcher_PreferenceDisabledChannel(t *testing.T) {
	t.Parallel()
	s1 := &stubSender{name: "email"}
	s2 := &stubSender{name: "sms"}
	prefs := &stubPrefs{disabled: map[string]bool{"sms": true}}
	d := NewDispatcher([]Sender{s1, s2}, prefs)

	d.DispatchSync(context.Background(), testApproval(), testRecipient())

	if s1.callCount() != 1 {
		t.Errorf("email: expected 1 call, got %d", s1.callCount())
	}
	if s2.callCount() != 0 {
		t.Errorf("sms: expected 0 calls (disabled), got %d", s2.callCount())
	}
}

func TestDispatcher_PreferenceCheckError_DefaultsToSending(t *testing.T) {
	t.Parallel()
	s := &stubSender{name: "email"}
	prefs := &stubPrefs{err: errors.New("db connection lost")}
	d := NewDispatcher([]Sender{s}, prefs)

	d.DispatchSync(context.Background(), testApproval(), testRecipient())

	// On preference check error, we default to sending.
	if s.callCount() != 1 {
		t.Errorf("expected 1 call (default to send on error), got %d", s.callCount())
	}
}

func TestDispatcher_MissingPreferenceDefaultsEnabled(t *testing.T) {
	t.Parallel()
	s := &stubSender{name: "web-push"}
	// Empty disabled map → all channels default to enabled.
	prefs := &stubPrefs{disabled: map[string]bool{}}
	d := NewDispatcher([]Sender{s}, prefs)

	d.DispatchSync(context.Background(), testApproval(), testRecipient())

	if s.callCount() != 1 {
		t.Errorf("expected 1 call (missing preference defaults to enabled), got %d", s.callCount())
	}
}

func TestDispatch_IsNonBlocking(t *testing.T) {
	t.Parallel()
	// Verify the async Dispatch returns immediately even when senders
	// would block. This is the critical property for HTTP handlers.
	blocker := make(chan struct{})
	sendDone := make(chan struct{})
	s := &stubSender{name: "email", blocked: blocker, done: sendDone}
	d := NewDispatcher([]Sender{s}, nil)

	returned := make(chan struct{})
	go func() {
		d.Dispatch(context.Background(), testApproval(), testRecipient())
		close(returned)
	}()

	// Dispatch must return before the sender is unblocked.
	select {
	case <-returned:
		// Good — Dispatch returned while sender is still blocked.
	case <-time.After(2 * time.Second):
		t.Fatal("Dispatch blocked waiting for sender — should be async")
	}

	// Unblock the sender so the background goroutine finishes cleanly.
	close(blocker)

	// Wait for the background send to complete (with timeout to avoid hanging).
	select {
	case <-sendDone:
	case <-time.After(2 * time.Second):
		t.Fatal("background sender did not complete after unblocking")
	}

	if s.callCount() != 1 {
		t.Errorf("sender should have been called once after unblocking, got %d", s.callCount())
	}
}
