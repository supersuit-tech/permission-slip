package notify

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// stubGate is a test double for SMSGate.
type stubGate struct {
	allowed     bool
	checkErr    error
	recordErr   error
	mu          sync.Mutex
	recordCalls int
}

func (g *stubGate) CanSendSMS(_ context.Context, _ string) (bool, error) {
	if g.checkErr != nil {
		return false, g.checkErr
	}
	return g.allowed, nil
}

func (g *stubGate) RecordSMSSent(_ context.Context, _ string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recordCalls++
	return g.recordErr
}

func (g *stubGate) recordCallCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.recordCalls
}

func TestPlanGatedSender_FreeTierBlocked(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms"}
	gate := &stubGate{allowed: false}
	s := NewPlanGatedSender(inner, gate)

	err := s.Send(context.Background(), testApproval(), testRecipient())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if inner.callCount() != 0 {
		t.Errorf("expected inner sender NOT to be called (free tier), got %d calls", inner.callCount())
	}
	if gate.recordCallCount() != 0 {
		t.Errorf("expected no usage recording for free tier, got %d calls", gate.recordCallCount())
	}
}

func TestPlanGatedSender_PaidTierAllowed(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms"}
	gate := &stubGate{allowed: true}
	s := NewPlanGatedSender(inner, gate)

	err := s.Send(context.Background(), testApproval(), testRecipient())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if inner.callCount() != 1 {
		t.Errorf("expected inner sender to be called once (paid tier), got %d calls", inner.callCount())
	}
	if gate.recordCallCount() != 1 {
		t.Errorf("expected usage recorded once, got %d calls", gate.recordCallCount())
	}
}

func TestPlanGatedSender_PlanCheckError_SkipsSMS(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms"}
	gate := &stubGate{checkErr: errors.New("db unavailable")}
	s := NewPlanGatedSender(inner, gate)

	err := s.Send(context.Background(), testApproval(), testRecipient())
	if err != nil {
		t.Fatalf("expected nil error on plan check failure, got %v", err)
	}
	if inner.callCount() != 0 {
		t.Errorf("expected inner sender NOT called on plan check error, got %d calls", inner.callCount())
	}
}

func TestPlanGatedSender_InnerSendError_DoesNotTrackUsage(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms", err: errors.New("twilio error")}
	gate := &stubGate{allowed: true}
	s := NewPlanGatedSender(inner, gate)

	err := s.Send(context.Background(), testApproval(), testRecipient())
	if err == nil {
		t.Fatal("expected error from inner sender")
	}
	if gate.recordCallCount() != 0 {
		t.Errorf("expected no usage recording on send failure, got %d calls", gate.recordCallCount())
	}
}

func TestPlanGatedSender_RecordError_DoesNotFailSend(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms"}
	gate := &stubGate{allowed: true, recordErr: errors.New("db write failed")}
	s := NewPlanGatedSender(inner, gate)

	err := s.Send(context.Background(), testApproval(), testRecipient())
	if err != nil {
		t.Fatalf("expected nil error (usage tracking is best-effort), got %v", err)
	}
	if inner.callCount() != 1 {
		t.Errorf("expected inner sender called once, got %d calls", inner.callCount())
	}
}

func TestPlanGatedSender_Name(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms"}
	gate := &stubGate{allowed: true}
	s := NewPlanGatedSender(inner, gate)

	if s.Name() != "sms" {
		t.Errorf("expected Name() = %q, got %q", "sms", s.Name())
	}
}

func TestNewPlanGatedSender_NilGate_ReturnsInner(t *testing.T) {
	t.Parallel()
	inner := &stubSender{name: "sms"}
	s := NewPlanGatedSender(inner, nil)

	// With nil gate, should return inner directly.
	if s != inner {
		t.Error("expected NewPlanGatedSender(inner, nil) to return inner directly")
	}
}
