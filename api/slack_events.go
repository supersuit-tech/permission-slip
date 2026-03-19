package api

import (
	"encoding/json"
	"io"
	"log/slog"
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
			logWarn(deps.Logger, r, "SlackEvents: signing secret not configured")
			http.Error(w, "slack events not configured", http.StatusServiceUnavailable)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxSlackEventBodyBytes))
		if err != nil {
			logError(deps.Logger, r, "SlackEvents: error reading body", "error", err)
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
			logWarn(deps.Logger, r, "SlackEvents: slack connector not registered")
			http.Error(w, "slack connector not available", http.StatusServiceUnavailable)
			return
		}

		eventSource, ok := conn.(connectors.EventSource)
		if !ok {
			logWarn(deps.Logger, r, "SlackEvents: slack connector does not implement EventSource")
			http.Error(w, "slack connector does not support events", http.StatusServiceUnavailable)
			return
		}

		envelope, err := eventSource.VerifyAndParseEvent(body, r.Header, deps.SlackSigningSecret)
		if err != nil {
			logWarn(deps.Logger, r, "SlackEvents: verification failed", "error", err)
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
		logInfo(deps.Logger, r, "SlackEvents: received event",
			"event_id", event.EventID, "event_type", event.EventType, "team_id", event.TeamID)

		// Deduplication: skip events we've already successfully processed.
		if dedup.isSeen(event.EventID) {
			logInfo(deps.Logger, r, "SlackEvents: event already processed, skipping",
				"event_id", event.EventID)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Dispatch to registered event handlers.
		if deps.EventBroker != nil {
			if err := deps.EventBroker.Dispatch(r.Context(), event); err != nil {
				logError(deps.Logger, r, "SlackEvents: handler error",
					"event_id", event.EventID, "error", err)
				CaptureError(r.Context(), err)
				// Do NOT mark as seen — return 500 so Slack retries.
				http.Error(w, "processing failed", http.StatusInternalServerError)
				return
			}
		}

		// Only mark as seen after successful dispatch, so Slack retries
		// are processed if the handler failed on the first attempt.
		dedup.markSeen(event.EventID)
		w.WriteHeader(http.StatusOK)
	}
}

// logInfo logs at info level using deps.Logger if available.
func logInfo(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	if logger != nil {
		logger.InfoContext(r.Context(), msg, append([]any{"trace_id", TraceID(r.Context())}, args...)...)
	}
}

// logWarn logs at warn level using deps.Logger if available.
func logWarn(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	if logger != nil {
		logger.WarnContext(r.Context(), msg, append([]any{"trace_id", TraceID(r.Context())}, args...)...)
	}
}

// logError logs at error level using deps.Logger if available.
func logError(logger *slog.Logger, r *http.Request, msg string, args ...any) {
	if logger != nil {
		logger.ErrorContext(r.Context(), msg, append([]any{"trace_id", TraceID(r.Context())}, args...)...)
	}
}

// eventDeduplicator tracks recently processed event IDs to prevent
// duplicate processing when Slack retries delivery.
//
// TODO: This is a per-process in-memory store. For multi-instance deployments,
// replace with a shared store (Redis SET NX EX or DB INSERT ON CONFLICT) to
// prevent duplicate processing across pods. For single-instance Phase 2
// deployments, the in-memory approach is sufficient.
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

// isSeen returns true if the event ID has been seen within the TTL window.
// It does NOT record the event — call markSeen after successful processing.
func (d *eventDeduplicator) isSeen(eventID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Lazy cleanup: remove expired entries.
	for id, ts := range d.seen {
		if now.Sub(ts) > d.ttl {
			delete(d.seen, id)
		}
	}

	_, ok := d.seen[eventID]
	return ok
}

// markSeen records an event ID as successfully processed. Call this only
// after dispatch succeeds, so failed events can be retried by Slack.
func (d *eventDeduplicator) markSeen(eventID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[eventID] = time.Now()
}
