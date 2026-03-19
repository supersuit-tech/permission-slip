package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// maxSlackEventBodyBytes limits the event request body to 64 KB.
const maxSlackEventBodyBytes = 65536

// eventDeduplicationTTL is how long event IDs are remembered for dedup.
// Slack retries events for up to 60 minutes, so 1 hour provides full coverage.
const eventDeduplicationTTL = 1 * time.Hour

// RegisterSlackEventRoutes adds the Slack Events API webhook endpoint to the mux.
// Like the Stripe webhook, this lives outside /api/v1/ to bypass auth middleware.
// Slack authenticates requests via HMAC-SHA256 signature verification.
func RegisterSlackEventRoutes(mux *http.ServeMux, deps *Deps) {
	mux.Handle("POST /api/webhooks/slack/events", handleSlackEvent(deps))
}

func handleSlackEvent(deps *Deps) http.HandlerFunc {
	dedup := newEventDeduplicator(eventDeduplicationTTL)

	return func(w http.ResponseWriter, r *http.Request) {
		if deps.SlackSigningSecret == "" {
			log.Printf("[%s] SlackEvents: signing secret not configured", TraceID(r.Context()))
			http.Error(w, "slack events not configured", http.StatusServiceUnavailable)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxSlackEventBodyBytes))
		if err != nil {
			log.Printf("[%s] SlackEvents: error reading body: %v", TraceID(r.Context()), err)
			http.Error(w, "error reading body", http.StatusBadRequest)
			return
		}

		// Look up the Slack connector as an EventSource.
		if deps.Connectors == nil {
			http.Error(w, "connectors not available", http.StatusServiceUnavailable)
			return
		}
		conn, ok := deps.Connectors.Get("slack")
		if !ok {
			log.Printf("[%s] SlackEvents: slack connector not registered", TraceID(r.Context()))
			http.Error(w, "slack connector not available", http.StatusServiceUnavailable)
			return
		}

		eventSource, ok := conn.(connectors.EventSource)
		if !ok {
			log.Printf("[%s] SlackEvents: slack connector does not implement EventSource", TraceID(r.Context()))
			http.Error(w, "slack connector does not support events", http.StatusServiceUnavailable)
			return
		}

		envelope, err := eventSource.VerifyAndParseEvent(body, r.Header, deps.SlackSigningSecret)
		if err != nil {
			log.Printf("[%s] SlackEvents: verification failed: %v", TraceID(r.Context()), err)
			http.Error(w, "verification failed", http.StatusUnauthorized)
			return
		}

		// Handle URL verification challenge.
		if envelope.Challenge != "" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"challenge": envelope.Challenge})
			return
		}

		if envelope.Event == nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		event := envelope.Event
		log.Printf("[%s] SlackEvents: received event %s type=%s team=%s",
			TraceID(r.Context()), event.EventID, event.EventType, event.TeamID)

		// Deduplication: skip events we've already processed.
		if dedup.isDuplicate(event.EventID) {
			log.Printf("[%s] SlackEvents: event %s already processed, skipping", TraceID(r.Context()), event.EventID)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Dispatch to registered event handlers.
		if deps.EventBroker != nil {
			if err := deps.EventBroker.Dispatch(r.Context(), event); err != nil {
				log.Printf("[%s] SlackEvents: handler error for event %s: %v", TraceID(r.Context()), event.EventID, err)
				CaptureError(r.Context(), err)
				http.Error(w, "processing failed", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

// eventDeduplicator tracks recently processed event IDs to prevent
// duplicate processing when Slack retries delivery.
type eventDeduplicator struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
}

func newEventDeduplicator(ttl time.Duration) *eventDeduplicator {
	return &eventDeduplicator{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}
}

// isDuplicate returns true if the event ID has been seen within the TTL window.
// If not a duplicate, it records the event ID.
func (d *eventDeduplicator) isDuplicate(eventID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Lazy cleanup: remove expired entries.
	for id, ts := range d.seen {
		if now.Sub(ts) > d.ttl {
			delete(d.seen, id)
		}
	}

	if _, ok := d.seen[eventID]; ok {
		return true
	}
	d.seen[eventID] = now
	return false
}
