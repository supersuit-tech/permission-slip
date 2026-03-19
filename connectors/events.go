package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Event represents an inbound event from an external service.
type Event struct {
	ConnectorID string          `json:"connector_id"` // e.g., "slack"
	EventType   string          `json:"event_type"`   // e.g., "message.im"
	EventID     string          `json:"event_id"`     // service-specific ID for deduplication
	TeamID      string          `json:"team_id"`      // workspace/org identifier
	Timestamp   time.Time       `json:"timestamp"`
	Payload     json.RawMessage `json:"payload"` // event-type-specific data
}

// EventEnvelope wraps either a verification challenge response or a parsed event.
// Some services (e.g., Slack) require responding to a challenge during initial
// webhook URL registration. When Challenge is non-empty, the HTTP handler should
// respond with it directly instead of processing an event.
type EventEnvelope struct {
	Challenge string // non-empty for url_verification challenges
	Event     *Event // non-nil for normal events
}

// EventSource is optionally implemented by connectors that can receive
// inbound events from external services (e.g., Slack Events API, GitHub
// webhooks). The API layer checks for this interface when routing incoming
// webhook requests to the appropriate connector.
type EventSource interface {
	// VerifyAndParseEvent validates the authenticity of an incoming webhook
	// payload (e.g., HMAC signature verification) and parses it into an Event.
	//
	// The signingSecret is the connector-specific secret used for verification
	// (e.g., Slack signing secret, GitHub webhook secret).
	//
	// If the payload is a verification challenge, the returned EventEnvelope
	// has a non-empty Challenge field and nil Event.
	VerifyAndParseEvent(payload []byte, headers http.Header, signingSecret string) (*EventEnvelope, error)
}

// EventHandler processes inbound events from external services.
type EventHandler interface {
	HandleEvent(ctx context.Context, event *Event) error
}

// EventHandlerFunc adapts a plain function to the EventHandler interface.
type EventHandlerFunc func(ctx context.Context, event *Event) error

func (f EventHandlerFunc) HandleEvent(ctx context.Context, event *Event) error {
	return f(ctx, event)
}

// EventBroker manages event handler registration and dispatch.
// Handlers are registered by event type and called sequentially when
// an event of that type is dispatched.
type EventBroker struct {
	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

// NewEventBroker creates an empty event broker.
func NewEventBroker() *EventBroker {
	return &EventBroker{handlers: make(map[string][]EventHandler)}
}

// Subscribe registers a handler for the given event type.
func (b *EventBroker) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Dispatch sends an event to all handlers registered for its event type.
// Handlers are called sequentially. Returns the first error encountered.
func (b *EventBroker) Dispatch(ctx context.Context, event *Event) error {
	b.mu.RLock()
	handlers := b.handlers[event.EventType]
	b.mu.RUnlock()

	for _, h := range handlers {
		if err := h.HandleEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
