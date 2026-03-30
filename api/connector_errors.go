package api

import (
	"errors"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// handleConnectorError maps typed connector errors to HTTP responses and
// captures server-side errors to Sentry with connector-specific context.
// Returns true if the error was handled, false if the caller should handle it
// as an untyped (500) error.
//
// Connector error messages are surfaced to the caller so agents can see
// exactly what went wrong (e.g., "Slack channel not found", "GitHub API
// validation error: ..."). Connectors are responsible for crafting
// user-friendly, safe-to-expose messages.
func handleConnectorError(w http.ResponseWriter, r *http.Request, err error, cc ConnectorContext) bool {
	traceID := TraceID(r.Context())

	switch {
	case connectors.IsPaymentError(err):
		var pe *connectors.PaymentError
		connectors.AsPaymentError(err, &pe)
		switch pe.Code {
		case connectors.PaymentErrMissing, connectors.PaymentErrAmountRequired:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrPaymentMethodRequired, pe.Message))
		case connectors.PaymentErrNotFound:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrPaymentMethodNotFound, pe.Message))
		case connectors.PaymentErrPerTxLimit, connectors.PaymentErrMonthlyLimit:
			resp := newErrorResponse(ErrPaymentLimitExceeded, pe.Message, false)
			if pe.Details != nil {
				resp.Error.Details = pe.Details
			}
			RespondError(w, r, http.StatusForbidden, resp)
		case connectors.PaymentErrInvalidAmount:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, pe.Message))
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, pe.Message))
		}
		return true

	case connectors.IsValidationError(err):
		msg := "Validation failed"
		var ve *connectors.ValidationError
		if errors.As(err, &ve) && ve.Message != "" {
			msg = ve.Message
		}
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, msg))
		return true

	case connectors.IsRateLimitError(err):
		msg := "External service rate limited"
		var rl *connectors.RateLimitError
		retrySeconds := 0
		if connectors.AsRateLimitError(err, &rl) {
			retrySeconds = int(rl.RetryAfter.Seconds())
			if rl.Message != "" {
				msg = rl.Message
			}
		}
		log.Printf("[%s] connector rate limited: %v", traceID, err)
		CaptureConnectorError(r.Context(), err, cc)
		RespondError(w, r, http.StatusTooManyRequests, TooManyRequests(msg, retrySeconds))
		return true

	case connectors.IsExternalError(err):
		msg := "External service returned an error"
		var ee *connectors.ExternalError
		if errors.As(err, &ee) && ee.Message != "" {
			msg = ee.Message
		}
		log.Printf("[%s] connector external error: %v", traceID, err)
		CaptureConnectorError(r.Context(), err, cc)
		RespondError(w, r, http.StatusBadGateway, newErrorResponse(ErrUpstreamError, msg, true))
		return true

	case connectors.IsOAuthRefreshError(err):
		log.Printf("[%s] OAuth refresh error: %v", traceID, err)
		CaptureConnectorError(r.Context(), err, cc)
		var oauthErr *connectors.OAuthRefreshError
		msg := "OAuth authorization required — user must re-connect the provider in Settings"
		details := map[string]any{
			"action_required": "reauthorize",
		}
		if connectors.AsOAuthRefreshError(err, &oauthErr) {
			details["provider"] = oauthErr.Provider
			if oauthErr.Message != "" {
				msg = oauthErr.Message
			}
		}
		resp := newErrorResponse(ErrOAuthRefreshFailed, msg, false)
		resp.Error.Details = details
		RespondError(w, r, http.StatusUnauthorized, resp)
		return true

	case connectors.IsAuthError(err):
		msg := "External service rejected credentials"
		var ae *connectors.AuthError
		if errors.As(err, &ae) && ae.Message != "" {
			msg = ae.Message
		}
		log.Printf("[%s] connector auth error: %v", traceID, err)
		CaptureConnectorError(r.Context(), err, cc)
		RespondError(w, r, http.StatusBadGateway, newErrorResponse(ErrUpstreamError, msg, true))
		return true

	case connectors.IsTimeoutError(err):
		msg := "External service did not respond in time"
		var te *connectors.TimeoutError
		if errors.As(err, &te) && te.Message != "" {
			msg = te.Message
		}
		log.Printf("[%s] connector timeout: %v", traceID, err)
		CaptureConnectorError(r.Context(), err, cc)
		RespondError(w, r, http.StatusGatewayTimeout, newErrorResponse(ErrUpstreamError, msg, true))
		return true

	case connectors.IsCanceledError(err):
		msg := "Request was canceled"
		var ce *connectors.CanceledError
		if errors.As(err, &ce) && ce.Message != "" {
			msg = ce.Message
		}
		log.Printf("[%s] connector request canceled: %v", traceID, err)
		CaptureConnectorError(r.Context(), err, cc)
		// Canceled requests are NOT retryable — the caller intentionally canceled.
		RespondError(w, r, http.StatusBadGateway, newErrorResponse(ErrUpstreamError, msg, false))
		return true

	default:
		return false
	}
}
