package notify

import (
	"context"
	"log"
	"sync"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// BuildContext holds the runtime dependencies available to sender factories
// at startup. It is passed to each registered SenderFactory so factories can
// inspect config, access the database, and communicate results back.
type BuildContext struct {
	// DB is the database handle. Nil when no database is configured.
	DB db.DBTX

	// Config holds the notification configuration loaded from environment variables.
	Config Config

	// DevMode is true when the server is running in development mode.
	// Factories use this to relax requirements (e.g. auto-generate VAPID keys)
	// or degrade gracefully instead of calling log.Fatalf.
	DevMode bool

	// OnVAPIDPublicKey, when non-nil, is called by the web push factory once
	// VAPID keys are available. The caller stores the key for serving to
	// browser clients via the /push/vapid-public-key endpoint.
	OnVAPIDPublicKey func(publicKey string)
}

// SenderFactory constructs Senders from a BuildContext. It returns the senders
// to register (nil or empty when the channel is not configured) and any error
// encountered during construction. Errors are logged but do not prevent other
// factories from running.
type SenderFactory func(ctx context.Context, bc BuildContext) ([]Sender, error)

var (
	senderMu        sync.Mutex
	senderFactories []SenderFactory
)

// RegisterSenderFactory registers a SenderFactory to be called by BuildSenders.
// Factories are called in registration order. Packages register themselves in
// their init() function so registration happens automatically on import.
func RegisterSenderFactory(fn SenderFactory) {
	senderMu.Lock()
	defer senderMu.Unlock()
	senderFactories = append(senderFactories, fn)
}

// BuildSenders calls all registered factories with bc and returns the combined
// Sender slice. Errors from individual factories are logged but do not prevent
// other factories from running.
//
// To ensure all built-in senders are registered, import the notify/all package
// (or import sender packages individually) before calling BuildSenders.
func BuildSenders(ctx context.Context, bc BuildContext) []Sender {
	senderMu.Lock()
	factories := make([]SenderFactory, len(senderFactories))
	copy(factories, senderFactories)
	senderMu.Unlock()

	var senders []Sender
	for _, factory := range factories {
		ss, err := factory(ctx, bc)
		if err != nil {
			log.Printf("notify: failed to build sender: %v", err)
			continue
		}
		senders = append(senders, ss...)
	}
	return senders
}
