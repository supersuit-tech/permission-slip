package oauth

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// RefreshResult holds the new tokens and expiry from a successful token refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string // May differ from the original if the provider rotates refresh tokens.
	Expiry       time.Time
}

// RefreshTokens uses the provider's token endpoint to exchange a refresh token
// for a new access token. Returns the new tokens and expiry. The provider must
// have client credentials configured; the caller must have a valid refresh token.
//
// The context controls the HTTP timeout for the token exchange request.
func RefreshTokens(ctx context.Context, provider Provider, refreshToken string) (*RefreshResult, error) {
	if !provider.HasClientCredentials() {
		return nil, fmt.Errorf("oauth provider %q has no client credentials configured", provider.ID)
	}
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is empty for provider %q", provider.ID)
	}

	cfg := &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL:  provider.TokenURL,
			AuthStyle: provider.AuthStyle,
		},
	}

	// Create a token source from the existing refresh token.
	// oauth2.TokenSource will exchange the refresh token for a new access token.
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	ts := cfg.TokenSource(ctx, token)
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token exchange for provider %q: %w", provider.ID, err)
	}

	if newToken.AccessToken == "" {
		return nil, fmt.Errorf("provider %q returned an empty access token during refresh", provider.ID)
	}

	result := &RefreshResult{
		AccessToken: newToken.AccessToken,
		Expiry:      newToken.Expiry,
	}

	// Some providers rotate refresh tokens on each refresh. Use the new one
	// if provided, otherwise keep the original.
	if newToken.RefreshToken != "" {
		result.RefreshToken = newToken.RefreshToken
	} else {
		result.RefreshToken = refreshToken
	}

	// Slack v2_user token endpoint returns user tokens at the top level,
	// so no authed_user normalization is needed. The standard token
	// fields above already capture the refreshed user token correctly.

	return result, nil
}
