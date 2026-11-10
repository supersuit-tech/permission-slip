package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/supersuit-tech/permission-slip/db"
)

// oauthConnectionIdentityKey returns a stable identity string for deduplicating
// OAuth connections for the same user+provider. Per-instance providers (Shopify,
// Zendesk) key on instance identifiers so multiple stores remain allowed.
// Account-based providers (Google, GitHub, Microsoft) key on email when present.
func oauthConnectionIdentityKey(providerID string, extraData json.RawMessage) string {
	extra := oauthExtraDataMap(extraData)
	if len(extra) == 0 {
		return ""
	}
	if subdomain := strings.TrimSpace(extra["subdomain"]); subdomain != "" {
		return "subdomain:" + strings.ToLower(subdomain)
	}
	if shop := strings.TrimSpace(extra["shop_domain"]); shop != "" {
		return "shop:" + strings.ToLower(shop)
	}
	team := strings.TrimSpace(extra["team_name"])
	if providerID == "slack" && team != "" {
		return "slack_team:" + strings.ToLower(team)
	}
	email := strings.ToLower(strings.TrimSpace(extra["email"]))
	if email != "" {
		return "email:" + email
	}
	if providerID == "github" {
		if login := strings.ToLower(strings.TrimSpace(extra["display_name"])); login != "" {
			return "github_login:" + login
		}
	}
	return ""
}

func oauthExtraDataMap(extraData json.RawMessage) map[string]string {
	if len(extraData) == 0 {
		return nil
	}
	var extra map[string]string
	if err := json.Unmarshal(extraData, &extra); err != nil {
		return nil
	}
	return extra
}

// findOAuthConnectionIDsByIdentity returns connection IDs for the user and
// provider whose stored identity matches, newest first (same order as list API).
func findOAuthConnectionIDsByIdentity(
	ctx context.Context,
	tx db.DBTX,
	userID, providerID, identityKey string,
) ([]string, error) {
	if identityKey == "" {
		return nil, nil
	}
	conns, err := db.ListOAuthConnectionsByUser(ctx, tx, userID)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, c := range conns {
		if c.Provider != providerID {
			continue
		}
		if oauthConnectionIdentityKey(providerID, c.ExtraData) == identityKey {
			ids = append(ids, c.ID)
		}
	}
	return ids, nil
}

func purgeOAuthConnectionVaultSecrets(
	ctx context.Context,
	deps *Deps,
	tx db.DBTX,
	deleted *db.DeleteOAuthConnectionResult,
) error {
	if deleted == nil {
		return nil
	}
	if err := deps.Vault.DeleteSecret(ctx, tx, deleted.AccessTokenVaultID); err != nil {
		return err
	}
	if deleted.RefreshTokenVaultID != nil {
		if err := deps.Vault.DeleteSecret(ctx, tx, *deleted.RefreshTokenVaultID); err != nil {
			return err
		}
	}
	return nil
}
