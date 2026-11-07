package api

import (
	"errors"
	"strings"

	"github.com/supersuit-tech/permission-slip/agentwake"
	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

// normalizeOptionalCallbackURL trims and validates an optional per-request wake URL.
// Returns nil when raw is empty (no push wake for this request).
func normalizeOptionalCallbackURL(raw string) (*string, *ErrorResponse) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	provider, err := agentwake.ValidateCallbackURL(trimmed)
	if err != nil {
		var valErr *connectors.ValidationError
		if errors.As(err, &valErr) {
			er := BadRequest(ErrInvalidWebhookURL, valErr.Message)
			return nil, &er
		}
		er := BadRequest(ErrInvalidWebhookURL, err.Error())
		return nil, &er
	}
	_ = provider // provider is inferred again at delivery from the stored URL
	return &trimmed, nil
}

// approvalHasPushWakeDestination reports whether this approval stored a per-request callback URL.
func approvalHasPushWakeDestination(appr *db.Approval) bool {
	return appr != nil && appr.WakeCallbackURL != nil && strings.TrimSpace(*appr.WakeCallbackURL) != ""
}
