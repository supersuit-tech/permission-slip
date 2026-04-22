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
	"github.com/supersuit-tech/permission-slip/connectors"
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

func TestCaptureConnectorError_EnrichesTags(t *testing.T) {
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
	ctx = context.WithValue(ctx, traceIDKey{}, "trace_conn_123")

	CaptureConnectorError(ctx, &connectors.ExternalError{Message: "upstream 500"}, ConnectorContext{
		ActionType: "github.create_issue",
		AgentID:    42,
	})

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected at least one event to be captured")
	}

	event := events[0]
	checks := map[string]string{
		"trace_id":     "trace_conn_123",
		"action_type":  "github.create_issue",
		"connector_id": "github",
		"agent_id":     "42",
		"error_type":   "external",
	}
	for tag, want := range checks {
		if got := event.Tags[tag]; got != want {
			t.Errorf("tag %s = %q, want %q", tag, got, want)
		}
	}
}

func TestCaptureConnectorError_NoHubDoesNotPanic(t *testing.T) {
	t.Parallel()
	CaptureConnectorError(context.Background(), errors.New("test"), ConnectorContext{ActionType: "test.action"})
}

func TestCaptureConnectorError_ExplicitConnectorID(t *testing.T) {
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

	CaptureConnectorError(ctx, errors.New("list calendars"), ConnectorContext{
		ConnectorID: "google",
		ActionType:  "google.list_calendar_events",
		AgentID:     7,
	})

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected at least one event to be captured")
	}
	event := events[0]
	if got := event.Tags["connector_id"]; got != "google" {
		t.Errorf("connector_id tag = %q, want google", got)
	}
	if got := event.Tags["action_type"]; got != "google.list_calendar_events" {
		t.Errorf("action_type tag = %q", got)
	}
	if got := event.Tags["agent_id"]; got != "7" {
		t.Errorf("agent_id tag = %q, want 7", got)
	}
}

func TestClassifyConnectorError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{"external", &connectors.ExternalError{Message: "bad"}, "external"},
		{"auth", &connectors.AuthError{Message: "denied"}, "auth"},
		{"timeout", &connectors.TimeoutError{Message: "slow"}, "timeout"},
		{"rate_limit", &connectors.RateLimitError{Message: "429"}, "rate_limit"},
		{"oauth_refresh", &connectors.OAuthRefreshError{Provider: "google"}, "oauth_refresh"},
		{"validation", &connectors.ValidationError{Message: "bad param"}, "validation"},
		{"payment", &connectors.PaymentError{Code: connectors.PaymentErrMissing}, "payment"},
		{"canceled_error", &connectors.CanceledError{Message: "aborted"}, "canceled"},
		{"deadline_exceeded", context.DeadlineExceeded, "deadline_exceeded"},
		{"canceled", context.Canceled, "canceled"},
		{"unknown", errors.New("something weird"), "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyConnectorError(tc.err)
			if got != tc.want {
				t.Errorf("classifyConnectorError(%T) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestCaptureConnectorError_FingerprintSplitsByConnectorActionStatus(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	hub := sentry.NewHub(client, sentry.NewScope())
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	CaptureConnectorError(ctx, &connectors.ExternalError{StatusCode: 404, Message: "Chat app not found"}, ConnectorContext{
		ActionType: "google.chat_send_message",
		AgentID:    1,
	})

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected event to be captured")
	}
	fp := events[0].Fingerprint
	wantParts := []string{"connector", "external", "google", "google.chat_send_message", "404"}
	if len(fp) != len(wantParts) {
		t.Fatalf("fingerprint = %v, want %v", fp, wantParts)
	}
	for i, want := range wantParts {
		if fp[i] != want {
			t.Errorf("fingerprint[%d] = %q, want %q", i, fp[i], want)
		}
	}
}

func TestCaptureConnectorError_4xxExternalErrorIsWarning(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	hub := sentry.NewHub(client, sentry.NewScope())
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	CaptureConnectorError(ctx, &connectors.ExternalError{StatusCode: 400, Message: "bad range"}, ConnectorContext{
		ConnectorID: "google",
		ActionType:  "google.sheets_read_range",
	})

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected event to be captured")
	}
	if got := events[0].Level; got != sentry.LevelWarning {
		t.Errorf("level = %q, want %q (4xx external errors should not page)", got, sentry.LevelWarning)
	}
}

func TestCaptureConnectorError_5xxExternalErrorStaysError(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	hub := sentry.NewHub(client, sentry.NewScope())
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	CaptureConnectorError(ctx, &connectors.ExternalError{StatusCode: 503, Message: "upstream down"}, ConnectorContext{
		ConnectorID: "google",
		ActionType:  "google.sheets_read_range",
	})

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected event to be captured")
	}
	if got := events[0].Level; got == sentry.LevelWarning {
		t.Errorf("level = %q, expected default error-level (5xx is a real outage)", got)
	}
}

func TestCaptureConnectorError_OAuthRefreshIsWarning(t *testing.T) {
	t.Parallel()

	transport := &sentryTestTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://key@sentry.example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatalf("failed to create sentry client: %v", err)
	}

	hub := sentry.NewHub(client, sentry.NewScope())
	ctx := sentry.SetHubOnContext(context.Background(), hub)

	CaptureConnectorError(ctx, &connectors.OAuthRefreshError{
		Provider: "slack",
		Message:  "OAuth connection for \"slack\" has status \"needs_reauth\" — user must re-authorize",
	}, ConnectorContext{
		ConnectorID: "slack",
		ActionType:  "slack.list_channels",
		AgentID:     3,
	})

	events := transport.getEvents()
	if len(events) == 0 {
		t.Fatal("expected event to be captured")
	}
	if got := events[0].Level; got != sentry.LevelWarning {
		t.Errorf("level = %q, want %q (oauth_refresh is user-action-required, should not page)", got, sentry.LevelWarning)
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
func (t *sentryTestTransport) Flush(_ time.Duration) bool              { return true }
func (t *sentryTestTransport) FlushWithContext(_ context.Context) bool { return true }
func (t *sentryTestTransport) Close()                                  {}

// getEvents returns a snapshot of captured events (safe for concurrent use).
func (t *sentryTestTransport) getEvents() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	cp := make([]*sentry.Event, len(t.events))
	copy(cp, t.events)
	return cp
}
