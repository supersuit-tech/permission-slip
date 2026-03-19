package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestEventBroker_Dispatch(t *testing.T) {
	broker := NewEventBroker()

	var received *Event
	broker.Subscribe("message.im", EventHandlerFunc(func(_ context.Context, event *Event) error {
		received = event
		return nil
	}))

	event := &Event{
		ConnectorID: "slack",
		EventType:   "message.im",
		EventID:     "Ev01234",
		TeamID:      "T01234",
		Timestamp:   time.Now(),
		Payload:     json.RawMessage(`{"user":"U01234"}`),
	}

	if err := broker.Dispatch(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received == nil {
		t.Fatal("expected handler to be called")
	}
	if received.EventID != "Ev01234" {
		t.Errorf("expected event_id %q, got %q", "Ev01234", received.EventID)
	}
}

func TestEventBroker_NoHandlers(t *testing.T) {
	broker := NewEventBroker()

	event := &Event{
		EventType: "unregistered.type",
	}

	// Should not error when no handlers are registered.
	if err := broker.Dispatch(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEventBroker_HandlerError(t *testing.T) {
	broker := NewEventBroker()

	handlerErr := errors.New("handler failed")
	broker.Subscribe("test.event", EventHandlerFunc(func(_ context.Context, _ *Event) error {
		return handlerErr
	}))

	event := &Event{EventType: "test.event"}

	err := broker.Dispatch(context.Background(), event)
	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected handler error, got: %v", err)
	}
}

func TestEventBroker_MultipleHandlers(t *testing.T) {
	broker := NewEventBroker()

	var order []int
	broker.Subscribe("test.event", EventHandlerFunc(func(_ context.Context, _ *Event) error {
		order = append(order, 1)
		return nil
	}))
	broker.Subscribe("test.event", EventHandlerFunc(func(_ context.Context, _ *Event) error {
		order = append(order, 2)
		return nil
	}))

	event := &Event{EventType: "test.event"}
	if err := broker.Dispatch(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected handlers called in order [1,2], got %v", order)
	}
}
