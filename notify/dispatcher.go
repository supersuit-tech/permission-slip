package notify

import (
	"context"
	"log"
	"sync"

	"github.com/getsentry/sentry-go"
)

// PreferenceChecker determines whether a channel is enabled for a given user.
// The Dispatcher calls this before sending; if it returns false the channel
// is skipped. A nil PreferenceChecker means all channels are enabled.
type PreferenceChecker interface {
	// IsChannelEnabled returns true when the given channel should deliver
	// notifications to this user. Missing preference rows default to enabled.
	IsChannelEnabled(ctx context.Context, userID, channel string) (bool, error)
}

// Dispatcher fans out notifications to every registered Sender whose channel
// is enabled for the recipient. It is nil-safe: calling Dispatch on a nil
// *Dispatcher is a no-op.
type Dispatcher struct {
	senders []Sender
	prefs   PreferenceChecker
}

// NewDispatcher creates a Dispatcher with the provided senders and an optional
// preference checker. Pass nil for prefs to skip preference filtering (all
// channels will fire).
func NewDispatcher(senders []Sender, prefs PreferenceChecker) *Dispatcher {
	return &Dispatcher{senders: senders, prefs: prefs}
}

// Dispatch sends the notification to all enabled channels in the background.
// It returns immediately and never blocks the caller — this is critical so
// HTTP handlers don't wait on external delivery (email, SMS, push).
//
// Preference checks run synchronously on the caller's goroutine (they need
// the request-scoped DB connection), but the actual Send calls are detached
// and use context.Background() so delivery outlives the HTTP request.
//
// Calling Dispatch on a nil *Dispatcher is safe and does nothing.
func (d *Dispatcher) Dispatch(ctx context.Context, approval Approval, recipient Recipient) {
	if d == nil || len(d.senders) == 0 {
		return
	}

	toSend := d.enabledSenders(ctx, recipient)
	if len(toSend) == 0 {
		return
	}

	// Fire-and-forget: detach from request context so delivery outlives
	// the HTTP response.
	go d.sendAll(context.Background(), toSend, approval, recipient)
}

// DispatchSync is like Dispatch but blocks until all senders complete.
// Use this in tests or batch jobs where you need delivery to finish before
// proceeding. Production HTTP handlers should always use Dispatch.
func (d *Dispatcher) DispatchSync(ctx context.Context, approval Approval, recipient Recipient) {
	if d == nil || len(d.senders) == 0 {
		return
	}

	toSend := d.enabledSenders(ctx, recipient)
	d.sendAll(ctx, toSend, approval, recipient)
}

// enabledSenders returns the subset of senders whose channel is enabled for
// the recipient. Runs on the caller's goroutine so preference checks can
// use the request context.
func (d *Dispatcher) enabledSenders(ctx context.Context, recipient Recipient) []Sender {
	var toSend []Sender
	for _, s := range d.senders {
		// Skip channels where the recipient lacks the required contact info.
		if !recipientHasContact(s.Name(), recipient) {
			log.Printf("notify: channel %q skipped for user %s — missing contact info", s.Name(), recipient.UserID)
			continue
		}
		if d.prefs != nil {
			enabled, err := d.prefs.IsChannelEnabled(ctx, recipient.UserID, s.Name())
			if err != nil {
				log.Printf("notify: failed to check preference for channel %q user %s: %v", s.Name(), recipient.UserID, err)
				sentry.CaptureException(err)
				// On error, default to sending — better to over-notify than miss.
			} else if !enabled {
				log.Printf("notify: channel %q disabled for user %s, skipping", s.Name(), recipient.UserID)
				continue
			}
		}
		toSend = append(toSend, s)
	}
	return toSend
}

// recipientHasContact returns true if the recipient has the contact info
// required by the given channel. Channels not listed here are assumed to
// not need any specific contact field (e.g. web-push uses the UserID).
func recipientHasContact(channel string, r Recipient) bool {
	switch channel {
	case "email":
		return r.Email != nil && *r.Email != ""
	case "sms":
		return r.Phone != nil && *r.Phone != ""
	default:
		return true
	}
}

// sendAll delivers to every sender concurrently and waits for all to finish.
func (d *Dispatcher) sendAll(ctx context.Context, senders []Sender, approval Approval, recipient Recipient) {
	var wg sync.WaitGroup
	for _, sender := range senders {
		wg.Add(1)
		go func(s Sender) {
			defer wg.Done()
			if err := s.Send(ctx, approval, recipient); err != nil {
				log.Printf("notify: channel %q failed for approval %s: %v", s.Name(), approval.ApprovalID, err)
				hub := sentry.CurrentHub().Clone()
				hub.Scope().SetTag("channel", s.Name())
				hub.Scope().SetTag("approval_id", approval.ApprovalID)
				hub.CaptureException(err)
			}
		}(sender)
	}
	wg.Wait()
}
