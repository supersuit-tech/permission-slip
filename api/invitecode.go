package api

import (
	"fmt"
	"net/url"
)

const (
	// inviteCodeLen is the number of random characters in an invite code (PS-XXXX-XXXX = 8).
	inviteCodeLen = 8
)

// generateInviteCode produces a code in the format PS-XXXX-XXXX using a
// cryptographically random source and the safe character set.
func generateInviteCode() (string, error) {
	raw, err := generateRandomCode(inviteCodeLen)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("PS-%s-%s", raw[:4], raw[4:]), nil
}

// buildInviteURL constructs a convenience URL from the base URL and invite code.
// Returns "" if the base URL is unparseable or not absolute (missing scheme/host).
func buildInviteURL(baseURL, inviteCode string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	if u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.Path = "/invite/" + inviteCode
	return u.String()
}
