package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ApprovalEventBroker manages SSE connections for real-time approval events.
// Approvers connect via GET /approvals/events and receive notifications when
// new approvals are created, resolved, or cancelled — eliminating the need for
// 5-second polling.
//
// Design:
//   - One broker per process, created once and stored in Deps.
//   - Subscribers are keyed by user ID (Supabase auth subject).
//   - Each subscriber gets a buffered channel; if the buffer is full,
//     the new event is dropped (slow clients don't block the broker).
//   - A heartbeat comment (": heartbeat\n\n") is sent every 30 seconds
//     to keep connections alive through proxies/load balancers.
type ApprovalEventBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan string]struct{} // userID → set of channels
}

// NewApprovalEventBroker creates a broker ready for use.
func NewApprovalEventBroker() *ApprovalEventBroker {
	return &ApprovalEventBroker{
		subscribers: make(map[string]map[chan string]struct{}),
	}
}

// Subscribe registers a new SSE channel for the given user. The caller must
// call Unsubscribe when the connection closes.
func (b *ApprovalEventBroker) Subscribe(userID string) chan string {
	ch := make(chan string, 16)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = make(map[chan string]struct{})
	}
	b.subscribers[userID][ch] = struct{}{}
	return ch
}

// SubscribeWithLimit atomically checks the per-user connection limit and
// subscribes if under the limit. Returns (channel, true) on success, or
// (nil, false) if the limit would be exceeded. This avoids a TOCTOU race
// between checking SubscriberCount and calling Subscribe.
func (b *ApprovalEventBroker) SubscribeWithLimit(userID string, limit int) (chan string, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.subscribers[userID]) >= limit {
		return nil, false
	}
	ch := make(chan string, 16)
	if b.subscribers[userID] == nil {
		b.subscribers[userID] = make(map[chan string]struct{})
	}
	b.subscribers[userID][ch] = struct{}{}
	return ch, true
}

// Unsubscribe removes and closes the channel. Safe to call multiple times;
// a second call for the same channel is a no-op.
func (b *ApprovalEventBroker) Unsubscribe(userID string, ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subs, ok := b.subscribers[userID]; ok {
		if _, exists := subs[ch]; exists {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(b.subscribers, userID)
			}
			close(ch)
		}
	}
}

// SubscriberCount returns the number of active SSE connections for a user.
func (b *ApprovalEventBroker) SubscriberCount(userID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers[userID])
}

// maxSSEConnectionsPerUser is the maximum number of concurrent SSE connections
// a single user may hold. This prevents resource exhaustion from buggy clients
// or malicious actors opening many connections.
const maxSSEConnectionsPerUser = 10

// Notify sends an event string to all subscribers for the given user.
// Non-blocking: if a subscriber's buffer is full, the event is dropped.
func (b *ApprovalEventBroker) Notify(userID string, event string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers[userID] {
		select {
		case ch <- event:
		default:
			// Subscriber too slow — drop the event. The client will
			// catch up on next poll or reconnect.
		}
	}
}

// NotifyAll sends an event to every connected subscriber (all users).
// Used for broadcast events like server shutdown notices.
func (b *ApprovalEventBroker) NotifyAll(event string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subs := range b.subscribers {
		for ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
}

// RegisterApprovalEventRoutes adds the SSE endpoint for real-time approval updates.
//
// EventSource (the browser SSE API) does not support custom headers, so the
// client passes the Supabase session token as a `?token=` query parameter.
// This middleware copies it into the Authorization header before the standard
// RequireProfile middleware runs, keeping auth logic in one place.
func RegisterApprovalEventRoutes(mux *http.ServeMux, deps *Deps) {
	if deps.ApprovalEvents == nil {
		return
	}
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /approvals/events", sseTokenFromQuery(requireProfile(handleApprovalEvents(deps))))
}

// sseTokenFromQuery is a small middleware that copies a `?token=` query
// parameter into the Authorization header. This lets EventSource clients
// (which cannot set custom headers) authenticate with the standard JWT flow.
func sseTokenFromQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from query once so we can both use and then remove it.
		token := r.URL.Query().Get("token")

		// If no Authorization header is set, promote the token to a Bearer token.
		if r.Header.Get("Authorization") == "" && token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}

		// Remove the token query parameter to avoid leaking it via RequestURI logging.
		if token != "" {
			q := r.URL.Query()
			q.Del("token")
			r.URL.RawQuery = q.Encode()
			if r.URL.RawQuery != "" {
				r.RequestURI = r.URL.Path + "?" + r.URL.RawQuery
			} else {
				r.RequestURI = r.URL.Path
			}
		}

		next.ServeHTTP(w, r)
	})
}

func handleApprovalEvents(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Use http.ResponseController to access Flush through middleware
		// wrappers (e.g. statusRecorder) that implement Unwrap().
		// A direct w.(http.Flusher) type assertion fails on wrapped writers
		// because the wrapper type doesn't implement the interface, even
		// though the underlying writer does.
		rc := http.NewResponseController(w)

		profile := Profile(r.Context())
		userID := profile.ID

		// Atomically check per-user connection limit and subscribe.
		// Using SubscribeWithLimit avoids a TOCTOU race between checking
		// the count and subscribing.
		ch, ok := deps.ApprovalEvents.SubscribeWithLimit(userID, maxSSEConnectionsPerUser)
		if !ok {
			RespondError(w, r, http.StatusTooManyRequests,
				TooManyRequests("Too many concurrent SSE connections. Close unused tabs and retry.", 5))
			return
		}
		defer deps.ApprovalEvents.Unsubscribe(userID, ch)

		// Set SSE headers.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

		// Send an initial connected event so the client knows the stream is live.
		fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
		if err := rc.Flush(); err != nil {
			return
		}

		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				fmt.Fprintf(w, "%s\n\n", event)
				rc.Flush()
			case <-heartbeat.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				rc.Flush()
			}
		}
	}
}

// notifyApprovalChange sends an SSE event to the approver when an approval
// changes state. Called from approval handlers (create, approve, deny, cancel).
func notifyApprovalChange(deps *Deps, userID string, eventType string, approvalID string) {
	if deps.ApprovalEvents == nil {
		return
	}
	event := fmt.Sprintf("event: %s\ndata: {\"approval_id\":%q}", eventType, approvalID)
	deps.ApprovalEvents.Notify(userID, event)
	if deps.Logger != nil {
		deps.Logger.Debug("SSE event sent",
			slog.String("event_type", eventType),
			slog.String("approval_id", approvalID),
		)
	}
}

// notifyApprovalExecuted sends an SSE event with execution status when an
// approved action has been executed. This allows agents listening via SSE to
// receive execution results without polling.
func notifyApprovalExecuted(deps *Deps, userID string, approvalID string, executionStatus string) {
	if deps.ApprovalEvents == nil {
		return
	}
	event := fmt.Sprintf("event: approval_executed\ndata: {\"approval_id\":%q,\"execution_status\":%q}", approvalID, executionStatus)
	deps.ApprovalEvents.Notify(userID, event)
	if deps.Logger != nil {
		deps.Logger.Debug("SSE event sent",
			slog.String("event_type", "approval_executed"),
			slog.String("approval_id", approvalID),
			slog.String("execution_status", executionStatus),
		)
	}
}
