package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip/connectors"
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

// Severity aliases for CaptureMessage so callers don't have to depend on the
// sentry-go package directly.
const (
	SeverityInfo    = sentry.LevelInfo
	SeverityWarning = sentry.LevelWarning
	SeverityError   = sentry.LevelError
)

// CaptureMessage reports a non-error message (e.g. a security-relevant event
// like a failed confirmation code attempt) to Sentry at the given severity.
// The trace ID from ctx is attached as a tag so the message can be correlated
// with logs. If Sentry is not initialized (no DSN), this is a no-op.
func CaptureMessage(ctx context.Context, level sentry.Level, msg string) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	hub.WithScope(func(scope *sentry.Scope) {
		if traceID := TraceID(ctx); traceID != "" {
			scope.SetTag("trace_id", traceID)
		}
		scope.SetLevel(level)
		hub.CaptureMessage(msg)
	})
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
		errorType := classifyConnectorError(err)
		scope.SetTag("error_type", errorType)

		// Fingerprint by connector + action + error class (+ status for external
		// errors) so distinct upstream failures land in their own Sentry issues
		// instead of collapsing into one bucket keyed only by exception type +
		// stack trace. Without this, every ExternalError — regardless of which
		// connector, action, or status code — merges into a single issue,
		// making triage impossible.
		fp := []string{"connector", errorType}
		connID := cc.ConnectorID
		if connID == "" && cc.ActionType != "" {
			if derived := connectorIDFromActionType(cc.ActionType); derived != nil {
				connID = *derived
			}
		}
		if connID != "" {
			fp = append(fp, connID)
		}
		if cc.ActionType != "" {
			fp = append(fp, cc.ActionType)
		}
		var extErr *connectors.ExternalError
		if errors.As(err, &extErr) {
			fp = append(fp, strconv.Itoa(extErr.StatusCode))
			// 4xx from an external service means the request we sent was
			// rejected (bad params, missing resource, user-level config
			// problem). These are worth keeping in Sentry for visibility and
			// product-side follow-up, but they aren't backend outages and
			// shouldn't page via error-level alert rules. 5xx stays at the
			// default error level so genuine upstream outages still alert.
			if extErr.StatusCode >= 400 && extErr.StatusCode < 500 {
				scope.SetLevel(sentry.LevelWarning)
			}
		}
		// OAuth refresh failures are a user-action-required state, not a
		// backend failure: the user's token was revoked, expired without a
		// refresh token, or the connection was force-flipped to needs_reauth
		// by a scope-change migration. Keep it in Sentry for visibility but
		// don't page — same rationale as 4xx external errors above.
		if errorType == "oauth_refresh" {
			scope.SetLevel(sentry.LevelWarning)
		}
		scope.SetFingerprint(fp)

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
