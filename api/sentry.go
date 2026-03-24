package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip-web/connectors"
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

// ConnectorContext carries connector-specific metadata for Sentry error reports.
type ConnectorContext struct {
	ConnectorID string
	ActionType  string
	AgentID     int64
}

// CaptureConnectorError reports a connector execution error to Sentry with
// enriched tags: action_type, connector_id (derived from action_type),
// agent_id, and error_type. These tags enable filtering and grouping connector
// failures in the Sentry dashboard by connector, action, and error category.
func CaptureConnectorError(ctx context.Context, err error, cc ConnectorContext) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	hub.WithScope(func(scope *sentry.Scope) {
		if traceID := TraceID(ctx); traceID != "" {
			scope.SetTag("trace_id", traceID)
		}
		if cc.ActionType != "" {
			scope.SetTag("action_type", cc.ActionType)
		}
		if cc.ConnectorID != "" {
			scope.SetTag("connector_id", cc.ConnectorID)
		} else if cc.ActionType != "" {
			if connID := connectorIDFromActionType(cc.ActionType); connID != nil {
				scope.SetTag("connector_id", *connID)
			}
		}
		if cc.AgentID != 0 {
			scope.SetTag("agent_id", fmt.Sprintf("%d", cc.AgentID))
		}
		scope.SetTag("error_type", classifyConnectorError(err))
		hub.CaptureException(err)
	})
}

// classifyConnectorError returns a short label for the error type, used as
// the "error_type" Sentry tag for filtering and grouping.
//
// Case ordering matches handleConnectorError so both functions classify
// the same error identically if a future error wraps multiple types.
func classifyConnectorError(err error) string {
	switch {
	case connectors.IsPaymentError(err):
		return "payment"
	case connectors.IsValidationError(err):
		return "validation"
	case connectors.IsRateLimitError(err):
		return "rate_limit"
	case connectors.IsExternalError(err):
		return "external"
	case connectors.IsOAuthRefreshError(err):
		return "oauth_refresh"
	case connectors.IsAuthError(err):
		return "auth"
	case connectors.IsTimeoutError(err):
		return "timeout"
	case connectors.IsCanceledError(err):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "unknown"
	}
}
