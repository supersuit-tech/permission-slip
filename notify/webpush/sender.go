package webpush

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	wplib "github.com/SherClockHolmes/webpush-go"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
)

// Sender implements notify.Sender for the Web Push channel. It sends push
// messages to all registered browser subscriptions for a recipient.
type Sender struct {
	keys    *VAPIDKeys
	subject string // mailto: URL for VAPID — required by push services
	db      db.DBTX
}

// New creates a Web Push sender. The subject should be a mailto: URL
// (e.g. "mailto:admin@example.com") as required by the Web Push spec.
func New(keys *VAPIDKeys, subject string, d db.DBTX) *Sender {
	return &Sender{
		keys:    keys,
		subject: subject,
		db:      d,
	}
}

// Name returns "web-push" — matches the notification_preferences.channel value.
func (s *Sender) Name() string { return "web-push" }

// pushPayload is the JSON structure sent in the push message body.
// Kept minimal per spec: title, body, approval URL, and approval ID.
type pushPayload struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	URL        string `json:"url"`
	ApprovalID string `json:"approval_id"`
}

// Send delivers a push notification to all of the recipient's registered
// browser subscriptions. Expired subscriptions (410 Gone) are automatically
// cleaned up. Partial failures are logged but do not cause the overall
// Send to return an error — best-effort delivery across all devices.
func (s *Sender) Send(ctx context.Context, approval notify.Approval, recipient notify.Recipient) error {
	subs, err := db.ListWebPushSubscriptionsByUserID(ctx, s.db, recipient.UserID)
	if err != nil {
		return fmt.Errorf("list push subscriptions: %w", err)
	}
	if len(subs) == 0 {
		return nil
	}

	body := buildBody(approval)
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}

	ttlSeconds := int(time.Until(approval.ExpiresAt).Seconds())
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}

	opts := &wplib.Options{
		Subscriber:      s.subject,
		VAPIDPublicKey:  s.keys.PublicKey,
		VAPIDPrivateKey: s.keys.PrivateKey,
		TTL:             ttlSeconds,
		Urgency:         wplib.UrgencyHigh,
		Topic:           "approval-" + approval.ApprovalID,
	}

	var lastErr error
	for _, sub := range subs {
		// Web-push subscriptions always have these fields set (enforced by DB constraint).
		wpSub := &wplib.Subscription{
			Endpoint: derefStr(sub.Endpoint),
			Keys: wplib.Keys{
				P256dh: derefStr(sub.P256dh),
				Auth:   derefStr(sub.Auth),
			},
		}

		resp, err := wplib.SendNotificationWithContext(ctx, payload, wpSub, opts)
		if err != nil {
			log.Printf("webpush: failed to send to subscription %d: %v", sub.ID, err)
			lastErr = err
			continue
		}
		io.Copy(io.Discard, resp.Body) //nolint:errcheck // drain for connection reuse
		resp.Body.Close()

		if resp.StatusCode == http.StatusGone {
			// Subscription expired — clean up
			log.Printf("webpush: subscription %d expired (410 Gone), removing", sub.ID)
			if err := db.DeletePushSubscriptionByEndpoint(ctx, s.db, derefStr(sub.Endpoint)); err != nil {
				log.Printf("webpush: failed to delete expired subscription: %v", err)
			}
			continue
		}

		if resp.StatusCode >= 400 {
			log.Printf("webpush: push service returned %d for subscription %d", resp.StatusCode, sub.ID)
			lastErr = fmt.Errorf("push service returned %d", resp.StatusCode)
			continue
		}
	}

	return lastErr
}

// derefStr returns the string value of a pointer, or "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildBody constructs the push notification payload from the approval data.
// Delegates to notify.BuildPushContent for consistent messaging across channels.
func buildBody(approval notify.Approval) pushPayload {
	c := notify.BuildPushContent(approval)
	return pushPayload{
		Title:      c.Title,
		Body:       c.Body,
		URL:        c.URL,
		ApprovalID: c.ApprovalID,
	}
}
