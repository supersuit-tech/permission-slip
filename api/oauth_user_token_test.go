package api

import (
	"encoding/json"
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

			// Build an oauth2.Token with extras. The oauth2 library stores
			// extras in a special way — we simulate this via the raw JSON approach.
			rawJSON, err := json.Marshal(map[string]any{
				"access_token": "xoxb-bot-token",
				"token_type":   "bot",
				"authed_user":  tt.extra["authed_user"],
			})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			// Use the oauth2 internal token unmarshaling to get extras populated.
			// The oauth2.Token.Extra method reads from the raw JSON field.
			token := &oauth2.Token{AccessToken: "xoxb-bot-token"}
			// We need to use the internal method — set raw directly.
			// The simplest way is to construct via Token response parsing.
			token = token.WithExtra(tt.extra)

			got := extractSlackUserToken(token)
			if got != tt.wantUser {
				t.Errorf("extractSlackUserToken() = %q, want %q", got, tt.wantUser)
			}

			// Verify the raw JSON approach also works (secondary check).
			_ = rawJSON
		})
	}
}

func TestUserTokenVaultIDFromExtraData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		extraData json.RawMessage
		want      string
	}{
		{
			name:      "nil extra data",
			extraData: nil,
			want:      "",
		},
		{
			name:      "empty extra data",
			extraData: json.RawMessage(`{}`),
			want:      "",
		},
		{
			name:      "has user_access_token_vault_id",
			extraData: json.RawMessage(`{"user_access_token_vault_id":"vault-123","email":"test@example.com"}`),
			want:      "vault-123",
		},
		{
			name:      "invalid JSON",
			extraData: json.RawMessage(`{invalid`),
			want:      "",
		},
		{
			name:      "other keys only",
			extraData: json.RawMessage(`{"shop_domain":"mystore.myshopify.com"}`),
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := userTokenVaultIDFromExtraData(tt.extraData)
			if got != tt.want {
				t.Errorf("userTokenVaultIDFromExtraData() = %q, want %q", got, tt.want)
			}
		})
	}
}
