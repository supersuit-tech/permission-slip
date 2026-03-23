package oauth

import (
	"time"

	"golang.org/x/oauth2"
)

// SlackAuthedUserAccessToken returns authed_user.access_token from a Slack
// oauth.v2.access response, or "" if absent.
func SlackAuthedUserAccessToken(tok *oauth2.Token) string {
	if tok == nil {
		return ""
	}
	authedUser, ok := tok.Extra("authed_user").(map[string]any)
	if !ok {
		return ""
	}
	s, _ := authedUser["access_token"].(string)
	return s
}

// NormalizeSlackUserOAuthToken rewrites a Slack oauth.v2.access (or refresh)
// token so AccessToken, RefreshToken, and Expiry match the authorizing user
// when authed_user is present.
//
// Slack nests user refresh_token and expires_in under authed_user when token
// rotation is enabled; the top-level refresh_token refers to the bot token.
// Without this normalization, we would store a user access_token with a bot
// refresh_token and refresh would fail or return the wrong token type.
func NormalizeSlackUserOAuthToken(tok *oauth2.Token) *oauth2.Token {
	if tok == nil {
		return nil
	}
	authedUser, ok := tok.Extra("authed_user").(map[string]any)
	if !ok {
		return tok
	}
	userAccess, _ := authedUser["access_token"].(string)
	if userAccess == "" {
		return tok
	}
	t := *tok
	t.AccessToken = userAccess
	if rt, ok := authedUser["refresh_token"].(string); ok && rt != "" {
		t.RefreshToken = rt
	}
	// If authed_user has no refresh_token, keep the top-level value (user-only
	// apps may issue the refresh at the root). When both bot and user tokens
	// are returned with rotation, Slack puts the user refresh under authed_user.
	if exp := slackAuthedUserExpiry(authedUser); exp != nil {
		t.Expiry = *exp
	}
	return &t
}

func slackAuthedUserExpiry(authedUser map[string]any) *time.Time {
	v, ok := authedUser["expires_in"]
	if !ok {
		return nil
	}
	var secs int64
	switch x := v.(type) {
	case float64:
		secs = int64(x)
	case int:
		secs = int64(x)
	case int64:
		secs = x
	default:
		return nil
	}
	if secs <= 0 {
		return nil
	}
	e := time.Now().Add(time.Duration(secs) * time.Second)
	return &e
}
