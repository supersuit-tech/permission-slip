package agentwake

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// InferProviderFromCallbackURL returns the wake provider for a per-request callback URL.
func InferProviderFromCallbackURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("callback_url is empty")
	}
	if err := ValidateGrokBotWebhookURL(trimmed); err == nil {
		return ProviderGrokBot, nil
	}
	if err := connectors.ValidatePrivateNetworkURL(trimmed, "callback_url"); err != nil {
		return "", err
	}
	return ProviderOpenClaw, nil
}

// ValidateCallbackURL checks a per-request wake callback URL and returns its provider.
func ValidateCallbackURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", fmt.Errorf("callback_url is empty")
	}
	if len(trimmed) > 2048 {
		return "", fmt.Errorf("callback_url exceeds maximum length of 2048 characters")
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("callback_url is not a valid URL")
	}
	if u.User != nil {
		return "", fmt.Errorf("callback_url must not include credentials")
	}
	return InferProviderFromCallbackURL(trimmed)
}
