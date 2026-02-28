package api

import (
	"context"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// SentryTraceIDMiddleware enriches the Sentry scope with request-level context:
// the trace ID (for log correlation), HTTP method, and URL path. It must run
// after TraceIDMiddleware (so the trace ID is available) and be nested inside
// sentryhttp.Handler (so a hub exists in the request context before tags are set).
func SentryTraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
			if traceID := TraceID(r.Context()); traceID != "" {
				hub.Scope().SetTag("trace_id", traceID)
			}
			hub.Scope().SetTag("http.method", r.Method)
			hub.Scope().SetTag("http.path", r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

// SetSentryUser sets the authenticated user ID on the Sentry scope so error
// events show which user was affected. Call this after authentication succeeds.
func SetSentryUser(ctx context.Context, userID string) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.Scope().SetUser(sentry.User{ID: userID})
	}
}

// CaptureError reports an error to Sentry with the trace ID from ctx
// attached as a tag. If Sentry is not initialized (no DSN), this is a no-op.
func CaptureError(ctx context.Context, err error) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	if traceID := TraceID(ctx); traceID != "" {
		hub.Scope().SetTag("trace_id", traceID)
	}
	hub.CaptureException(err)
}
