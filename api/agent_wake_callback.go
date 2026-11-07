package api

import (
	"context"
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
	_ = provider
	return &trimmed, nil
}

// ensureCallbackMatchesRegisteredProvider rejects a callback URL that does not
// match the agent's registered wake provider. With no token registered, the
// generic callback_url check is enough.
func ensureCallbackMatchesRegisteredProvider(ctx context.Context, deps *Deps, agentID int64, callbackURL string) *ErrorResponse {
	if deps == nil || deps.DB == nil || strings.TrimSpace(callbackURL) == "" {
		return nil
	}
	cfg, err := db.GetAgentWebhookConfig(ctx, deps.DB, agentID)
	if err != nil {
		er := InternalError("Failed to load wake credentials")
		return &er
	}
	if cfg == nil || cfg.WebhookTokenVaultID == nil || *cfg.WebhookTokenVaultID == "" {
		return nil
	}
	provider := cfg.WebhookProvider
	if provider == "" {
		provider = agentwake.ProviderOpenClaw
	}
	if err := validateWakeWebhookURL(provider, callbackURL); err != nil {
		var valErr *connectors.ValidationError
		if errors.As(err, &valErr) {
			er := BadRequest(ErrInvalidWebhookURL, valErr.Message)
			return &er
		}
		er := BadRequest(ErrInvalidWebhookURL, err.Error())
		return &er
	}
	return nil
}

// approvalHasPushWakeDestination reports whether this approval stored a per-request callback URL.
func approvalHasPushWakeDestination(appr *db.Approval) bool {
	return appr != nil && appr.WakeCallbackURL != nil && strings.TrimSpace(*appr.WakeCallbackURL) != ""
}
