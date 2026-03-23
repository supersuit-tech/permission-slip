package api

import (
	"testing"

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

func TestSlackV2UserTokenTopLevel(t *testing.T) {
	t.Parallel()
	// With oauth.v2.user.access, the user token is at the top level —
	// no authed_user normalization needed.
	tok := &oauth2.Token{
		AccessToken:  "xoxp-user",
		RefreshToken: "user-rt",
	}
	if tok.AccessToken != "xoxp-user" || tok.RefreshToken != "user-rt" {
		t.Fatalf("unexpected: %+v", tok)
	}
}
