// Package notify provides the notification dispatch infrastructure for
// Permission Slip. It defines the Sender interface that individual channels
// (email, web-push, SMS) implement, a Dispatcher that fans out to all
// configured senders, and the shared types (Approval, Recipient) that
// senders receive. Additional channels (for example, mobile push) can be
// added by implementing the Sender interface.
//
// Adding a new channel requires three steps:
//  1. Implement the Sender interface.
//  2. Add env-var loading for the channel's config.
//  3. Register the sender in main.go.
//
// No changes to the Dispatcher or approval handler are needed.
package notify

import (
	"context"
	"encoding/json"
	"time"
)

// Sender is the interface that each notification channel implements.
// Implementations must be safe for concurrent use.
type Sender interface {
	// Send delivers a notification for the given approval to the recipient.
	// Implementations should return promptly; long-running work (retries,
	// external HTTP calls) should be handled internally with appropriate
	// timeouts.
	Send(ctx context.Context, approval Approval, recipient Recipient) error

	// Name returns a stable, lowercase identifier for the channel
	// (e.g. "email", "web-push", "sms"). Used for logging and to match
	// against notification_preferences.channel.
	Name() string
}

// NotificationType distinguishes different kinds of notifications so
// templates and formatters can adapt their content. The zero value
// (empty string) means a standard approval notification.
type NotificationType string

const (
	// NotificationTypeApproval is the default — an approval request from an agent.
	NotificationTypeApproval NotificationType = ""
	// NotificationTypePaymentFailed is sent when a subscription payment fails.
	NotificationTypePaymentFailed NotificationType = "payment_failed"
)

// Approval holds the fields a notification channel needs to construct its
// message. It is deliberately decoupled from the db.Approval type so the
// notify package stays dependency-free of the database layer.
type Approval struct {
	ApprovalID  string
	AgentID     int64
	AgentName   string          // human-readable; may be empty
	Action      json.RawMessage // raw JSONB from the approvals table
	Context     json.RawMessage // raw JSONB from the approvals table
	ApprovalURL string          // deep link to the approval UI (or billing settings for payment failures)
	ExpiresAt   time.Time
	CreatedAt   time.Time
	Type        NotificationType // determines which email/SMS/push template to use; zero value = approval
}

// Recipient identifies who should receive the notification and provides
// the contact info that senders need.
type Recipient struct {
	UserID   string
	Username string
	Email    *string // nil when the user has not configured an email address
	Phone    *string // nil when the user has not configured a phone number
}
