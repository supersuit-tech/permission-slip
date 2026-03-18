package notify

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/getsentry/sentry-go"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// BuildContext holds the runtime dependencies available to sender factories
// at startup. It is passed to each registered SenderFactory so factories can
// inspect config, access the database, and communicate results back to the
// caller without requiring a global variable.
type BuildContext struct {
	// DB is the database handle. Nil when no database is configured (e.g.
	// during unit tests or when DATABASE_URL is not set). Factories that
	// require database access (web push, mobile push, SMS plan gating) should
	// return nil, nil when DB is nil rather than failing.
	DB db.DBTX

	// Config holds the notification configuration loaded from environment
	// variables. Each channel reads its own fields from Config to decide
	// whether it is configured.
	Config Config

	// DevMode is true when the server is running in development mode
	// (MODE=development). Factories use this to relax hard requirements —
	// for example, the web push factory auto-generates VAPID keys in dev mode
	// rather than failing, and uses a default VAPID_SUBJECT.
	DevMode bool

	// OnVAPIDPublicKey, when non-nil, is called by the web push factory with
	// the VAPID public key once VAPID keys are initialised. The caller stores
	// the key for serving to browser clients via /push/vapid-public-key.
	// This callback pattern avoids a global variable for the public key while
	// keeping the factory self-contained.
	OnVAPIDPublicKey func(publicKey string)
}

// SenderFactory constructs Senders from a BuildContext.
//
// Return values:
//   - (senders, nil) — channel is active; senders are appended to the result.
//   - (nil, nil)     — channel is disabled (e.g. env vars not set); silently skipped.
//   - (nil, err)     — unexpected failure; error is logged and the channel is skipped.
//
// Factories must not call log.Fatalf or os.Exit — return an error instead so
// BuildSenders can log it and continue with other channels.
type SenderFactory func(ctx context.Context, bc BuildContext) ([]Sender, error)

// namedFactory pairs a human-readable channel name with its factory so that
// error and panic log messages identify which channel failed.
type namedFactory struct {
	channel string
	fn      SenderFactory
}

var (
	senderMu        sync.Mutex
	senderFactories []namedFactory
)

// RegisterSenderFactory registers a SenderFactory to be called by BuildSenders.
// channel is a short human-readable name (e.g. "email", "sms") used in log
// messages to identify which factory failed. Factories are called in
// registration order. Packages register themselves in their init() function so
// registration happens automatically on import.
// Panics if fn is nil so the mistake surfaces immediately at registration time.
func RegisterSenderFactory(channel string, fn SenderFactory) {
	if fn == nil {
		panic("notify: RegisterSenderFactory called with nil factory for channel " + channel)
	}
	senderMu.Lock()
	defer senderMu.Unlock()
	senderFactories = append(senderFactories, namedFactory{channel: channel, fn: fn})
}

// SenderFactoryCount returns the number of registered sender factories.
// Useful for tests and diagnostics.
func SenderFactoryCount() int {
	senderMu.Lock()
	defer senderMu.Unlock()
	return len(senderFactories)
}

// BuildSenders calls all registered factories with bc and returns the combined
// Sender slice. Errors from individual factories are logged but do not prevent
// other factories from running. Panics inside factories are caught and treated
// as errors so a misbehaving factory cannot crash the server at startup.
//
// To ensure all built-in senders are registered, import the notify/all package
// (or import sender packages individually) before calling BuildSenders.
func BuildSenders(ctx context.Context, bc BuildContext) []Sender {
	senderMu.Lock()
	factories := make([]namedFactory, len(senderFactories))
	copy(factories, senderFactories)
	senderMu.Unlock()

	var senders []Sender
	for _, nf := range factories {
		ss, err := runFactory(ctx, bc, nf.fn)
		if err != nil {
			log.Printf("notify: %s sender factory failed: %v", nf.channel, err)
			sentry.CaptureException(err)
			continue
		}
		senders = append(senders, ss...)
	}
	return senders
}

// runFactory calls a single SenderFactory and recovers from panics.
func runFactory(ctx context.Context, bc BuildContext, fn SenderFactory) (ss []Sender, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn(ctx, bc)
}
