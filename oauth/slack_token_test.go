package oauth

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSlackAuthedUserAccessToken(t *testing.T) {
	t.Parallel()
	tok := (&oauth2.Token{AccessToken: "xoxb-bot"}).WithExtra(map[string]any{
		"authed_user": map[string]any{"access_token": "xoxp-user"},
	})
	if got := SlackAuthedUserAccessToken(tok); got != "xoxp-user" {
		t.Errorf("got %q, want xoxp-user", got)
	}
	if SlackAuthedUserAccessToken(nil) != "" {
		t.Error("nil token should return empty")
	}
}

func TestNormalizeSlackUserOAuthToken_UserRotation(t *testing.T) {
	t.Parallel()
	tok := (&oauth2.Token{
		AccessToken:  "xoxe.xoxb-bot",
		RefreshToken: "xoxe-bot-refresh",
	}).WithExtra(map[string]any{
		"authed_user": map[string]any{
			"access_token":  "xoxe.xoxp-user",
			"refresh_token": "xoxe-user-refresh",
			"expires_in":    float64(3600),
		},
	})
	out := NormalizeSlackUserOAuthToken(tok)
	if out.AccessToken != "xoxe.xoxp-user" {
		t.Errorf("AccessToken = %q", out.AccessToken)
	}
	if out.RefreshToken != "xoxe-user-refresh" {
		t.Errorf("RefreshToken = %q, want user refresh", out.RefreshToken)
	}
	if out.Expiry.IsZero() {
		t.Fatal("expected Expiry from authed_user.expires_in")
	}
	if d := time.Until(out.Expiry); d < 3500*time.Second || d > 3700*time.Second {
		t.Errorf("Expiry not ~1h from now: %v", d)
	}
}

func TestNormalizeSlackUserOAuthToken_UserOnlyKeepsTopLevelRefresh(t *testing.T) {
	t.Parallel()
	tok := (&oauth2.Token{
		AccessToken:  "xoxb-ignored",
		RefreshToken: "top-refresh",
	}).WithExtra(map[string]any{
		"authed_user": map[string]any{
			"access_token": "xoxp-only",
		},
	})
	out := NormalizeSlackUserOAuthToken(tok)
	if out.AccessToken != "xoxp-only" || out.RefreshToken != "top-refresh" {
		t.Errorf("got access=%q refresh=%q", out.AccessToken, out.RefreshToken)
	}
}

func TestNormalizeSlackUserOAuthToken_NoAuthedUserPassthrough(t *testing.T) {
	t.Parallel()
	tok := &oauth2.Token{AccessToken: "plain"}
	if got := NormalizeSlackUserOAuthToken(tok); got != tok {
		t.Error("expected same pointer when no authed_user")
	}
}
