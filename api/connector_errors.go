package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// handleConnectorError maps typed connector errors to HTTP responses.
// Returns true if the error was handled, false if the caller should handle it
// as an untyped (500) error.
//
// Error details are logged server-side but not exposed to the caller to avoid
// leaking internal information (upstream URLs, service internals, etc.).
func handleConnectorError(w http.ResponseWriter, r *http.Request, err error) bool {
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
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrPaymentMethodRequired, pe.Message))
		default:
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, pe.Message))
		}
		return true

	case connectors.IsValidationError(err):
		// ValidationError messages are safe to surface — they describe
		// parameter/credential issues the caller can fix.
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
		return true

	case connectors.IsRateLimitError(err):
		var rl *connectors.RateLimitError
		retrySeconds := 0
		if connectors.AsRateLimitError(err, &rl) {
			retrySeconds = int(rl.RetryAfter.Seconds())
		}
		log.Printf("[%s] connector rate limited: %v", traceID, err)
		RespondError(w, r, http.StatusTooManyRequests, TooManyRequests("External service rate limited", retrySeconds))
		return true

	case connectors.IsExternalError(err):
		log.Printf("[%s] connector external error: %v", traceID, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusBadGateway, newErrorResponse(ErrUpstreamError, "External service returned an error", true))
		return true

	case connectors.IsOAuthRefreshError(err):
		log.Printf("[%s] OAuth refresh error: %v", traceID, err)
		var oauthErr *connectors.OAuthRefreshError
		msg := "OAuth authorization required — user must re-connect the provider in Settings"
		details := map[string]any{
			"action_required": "reauthorize",
		}
		if connectors.AsOAuthRefreshError(err, &oauthErr) {
			details["provider"] = oauthErr.Provider
		}
		resp := newErrorResponse(ErrOAuthRefreshFailed, msg, false)
		resp.Error.Details = details
		RespondError(w, r, http.StatusUnauthorized, resp)
		return true

	case connectors.IsAuthError(err):
		log.Printf("[%s] connector auth error: %v", traceID, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusBadGateway, newErrorResponse(ErrUpstreamError, "External service rejected credentials", true))
		return true

	case connectors.IsTimeoutError(err):
		log.Printf("[%s] connector timeout: %v", traceID, err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusGatewayTimeout, newErrorResponse(ErrUpstreamError, "External service did not respond in time", true))
		return true

	default:
		return false
	}
}
