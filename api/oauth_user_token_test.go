package api

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestExtractSlackUserToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		extra    map[string]any
		wantUser string
	}{
		{
			name: "has authed_user with access_token",
			extra: map[string]any{
				"authed_user": map[string]any{
					"id":           "U12345",
					"access_token": "xoxp-user-token-value",
					"scope":        "search:read",
					"token_type":   "user",
				},
			},
			wantUser: "xoxp-user-token-value",
		},
		{
			name:     "no authed_user field",
			extra:    map[string]any{},
			wantUser: "",
		},
		{
			name: "authed_user without access_token",
			extra: map[string]any{
				"authed_user": map[string]any{
					"id": "U12345",
				},
			},
			wantUser: "",
		},
		{
			name: "authed_user is not a map",
			extra: map[string]any{
				"authed_user": "string-value",
			},
			wantUser: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := &oauth2.Token{AccessToken: "xoxb-bot-token"}
			token = token.WithExtra(tt.extra)

			got := extractSlackUserToken(token)
			if got != tt.wantUser {
				t.Errorf("extractSlackUserToken() = %q, want %q", got, tt.wantUser)
			}
		})
	}
}

func TestSlackUserTokenAsPrimary(t *testing.T) {
	t.Parallel()

	tok := (&oauth2.Token{AccessToken: "xoxb-app"}).WithExtra(map[string]any{
		"authed_user": map[string]any{
			"access_token": "xoxp-user",
		},
	})
	out := slackUserTokenAsPrimary(tok)
	if out.AccessToken != "xoxp-user" {
		t.Errorf("AccessToken = %q, want xoxp-user", out.AccessToken)
	}

	if slackUserTokenAsPrimary(nil) != nil {
		t.Error("nil input should return nil")
	}

	plain := &oauth2.Token{AccessToken: "only"}
	if got := slackUserTokenAsPrimary(plain); got != plain {
		t.Error("without authed_user, should return same token pointer")
	}
}
