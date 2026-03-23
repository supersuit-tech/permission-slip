package api

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/oauth"
	"golang.org/x/oauth2"
)

func TestExtractSlackTeamName(t *testing.T) {
	t.Parallel()
	tok := (&oauth2.Token{}).WithExtra(map[string]any{
		"team": map[string]any{"name": "Acme Corp"},
	})
	if got := extractSlackTeamName(tok); got != "Acme Corp" {
		t.Errorf("extractSlackTeamName() = %q", got)
	}
}

func TestOAuthSlackNormalizationUsesPackage(t *testing.T) {
	t.Parallel()
	// Regression: callback path must use oauth.NormalizeSlackUserOAuthToken for
	// user refresh_token alignment (see oauth/slack_token_test.go).
	tok := (&oauth2.Token{
		AccessToken:  "xoxb",
		RefreshToken: "bot-rt",
	}).WithExtra(map[string]any{
		"authed_user": map[string]any{
			"access_token":  "xoxp",
			"refresh_token": "user-rt",
		},
	})
	n := oauth.NormalizeSlackUserOAuthToken(tok)
	if n.AccessToken != "xoxp" || n.RefreshToken != "user-rt" {
		t.Fatalf("normalize: %+v", n)
	}
}
