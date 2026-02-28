package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestSentryTraceIDMiddleware_TagsHubWithTraceID(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	var capturedScope *sentry.Scope
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		hub := sentry.GetHubFromContext(r.Context())
		if hub != nil {
			capturedScope = hub.Scope()
		}
	})

	// Build the middleware chain: TraceID → (simulated sentryhttp) → SentryTraceID → inner.
	// Use a separate variable for the inner handler to avoid closure variable capture
	// causing infinite recursion when the outer handler calls itself.
	sentryTagged := SentryTraceIDMiddleware(inner)
	withHub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.NewHub(client, sentry.NewScope())
		ctx := sentry.SetHubOnContext(r.Context(), hub)
		sentryTagged.ServeHTTP(w, r.WithContext(ctx))
	})
	handler := TraceIDMiddleware(withHub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if capturedScope == nil {
		t.Fatal("expected scope to be captured")
	}

	// Apply scope to a test event to inspect the tags.
	event := &sentry.Event{Level: sentry.LevelInfo, Message: "probe"}
	capturedScope.ApplyToEvent(event, nil, client)

	tags := event.Tags
	if tags["trace_id"] == "" {
		t.Error("expected trace_id tag to be set on scope")
	}
	if tags["http.method"] != "GET" {
		t.Errorf("expected http.method tag = GET, got %q", tags["http.method"])
	}
	if tags["http.path"] != "/api/v1/agents" {
		t.Errorf("expected http.path tag = /api/v1/agents, got %q", tags["http.path"])
	}
}

func TestSentryTraceIDMiddleware_NoHubIsNoOp(t *testing.T) {
	t.Parallel()

	called := false
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		// No hub in context — should not panic.
		hub := sentry.GetHubFromContext(r.Context())
		if hub != nil {
			t.Error("expected no hub in context")
		}
	})

	handler := SentryTraceIDMiddleware(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called")
	}
}

func TestSetSentryUser_SetsUserOnHub(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	scope := sentry.NewScope()
	hub := sentry.NewHub(client, scope)
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	SetSentryUser(ctx, "user-abc-123")

	event := &sentry.Event{Level: sentry.LevelInfo, Message: "test"}
	scope.ApplyToEvent(event, nil, client)

	if event.User.ID != "user-abc-123" {
		t.Errorf("expected user ID = user-abc-123, got %q", event.User.ID)
	}
}

func TestSetSentryUser_NoHubIsNoOp(t *testing.T) {
	t.Parallel()

	// Should not panic when no hub is in context.
	SetSentryUser(context.Background(), "user-xyz")
}

func TestCaptureError_NoHubDoesNotPanic(t *testing.T) {
	t.Parallel()

	// CaptureError with a context that has no hub should not panic.
	// When Sentry is not initialized, CurrentHub.Clone() returns a hub
	// with a nil client, so CaptureException is a no-op.
	CaptureError(context.Background(), errors.New("test error"))
}

func TestCaptureError_WithTraceID(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	scope := sentry.NewScope()
	hub := sentry.NewHub(client, scope)
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	ctx = context.WithValue(ctx, traceIDKey{}, "trace_test123")

	CaptureError(ctx, errors.New("something broke"))

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected at least one event to be captured")
	}

	event := events[0]
	if event.Tags["trace_id"] != "trace_test123" {
		t.Errorf("expected trace_id tag = trace_test123, got %q", event.Tags["trace_id"])
	}
}

// sentryTestTransport captures events in memory for test assertions.
// A mutex guards the events slice in case the SDK delivers events from a
// background goroutine (defensive — our tests are single-goroutine but
// future SDK versions could change internal delivery).
type sentryTestTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *sentryTestTransport) Configure(_ sentry.ClientOptions) {}
func (t *sentryTestTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}
func (t *sentryTestTransport) Flush(_ time.Duration) bool             { return true }
func (t *sentryTestTransport) FlushWithContext(_ context.Context) bool { return true }
func (t *sentryTestTransport) Close()                                 {}

// getEvents returns a snapshot of captured events (safe for concurrent use).
func (t *sentryTestTransport) getEvents() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]*sentry.Event, len(t.events))
	copy(cp, t.events)
	return cp
}
