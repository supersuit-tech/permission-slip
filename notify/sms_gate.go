package notify

import (
	"context"
	"log"
)

// SMSGate determines whether a user's plan allows SMS notifications and
// records SMS usage after successful delivery. Implementations must be safe
// for concurrent use.
type SMSGate interface {
	// CanSendSMS returns true if the user's plan allows SMS notifications.
	CanSendSMS(ctx context.Context, userID string) (bool, error)

	// RecordSMSSent increments the SMS usage counter for the current billing period.
	RecordSMSSent(ctx context.Context, userID string) error
}

// PlanGatedSender wraps an SMS Sender with plan-based access control.
// Free-tier users are silently skipped (logged); paid-tier users have
// their SMS usage tracked after successful delivery.
type PlanGatedSender struct {
	inner Sender
	gate  SMSGate
}

// NewPlanGatedSender returns a Sender that checks the user's plan before
// delegating to the inner sender. Pass nil for gate to disable gating
// (all sends are allowed, no usage tracking).
func NewPlanGatedSender(inner Sender, gate SMSGate) Sender {
	if gate == nil {
		return inner
	}
	return &PlanGatedSender{inner: inner, gate: gate}
}

// Name returns the inner sender's channel name (e.g. "sms").
func (s *PlanGatedSender) Name() string { return s.inner.Name() }

// Send checks the user's plan before delivering. If the plan does not allow
// SMS, the send is skipped with a log message and nil is returned (so other
// channels are unaffected). On successful delivery, SMS usage is recorded.
func (s *PlanGatedSender) Send(ctx context.Context, approval Approval, recipient Recipient) error {
	allowed, err := s.gate.CanSendSMS(ctx, recipient.UserID)
	if err != nil {
		log.Printf("notify/sms: plan check failed for user %s: %v — skipping SMS", recipient.UserID, err)
		return nil
	}
	if !allowed {
		log.Printf("notify/sms: skipped for user %s (free tier)", recipient.UserID)
		return nil
	}

	if err := s.inner.Send(ctx, approval, recipient); err != nil {
		return err
	}

	if err := s.gate.RecordSMSSent(ctx, recipient.UserID); err != nil {
		log.Printf("notify/sms: failed to record usage for user %s: %v", recipient.UserID, err)
		// Don't fail the send — usage tracking is best-effort.
	}
	return nil
}
