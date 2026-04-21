package context

import (
	"fmt"
	"strings"
)

// BuildPermalink builds a deterministic Slack message permalink from the team
// domain (e.g. "acme" from acme.slack.com), channel ID, and message ts.
// No API call — used for approval context deep links.
func BuildPermalink(teamDomain, channelID, ts string) string {
	domain := strings.TrimSpace(teamDomain)
	domain = strings.TrimSuffix(domain, ".slack.com")
	domain = strings.TrimSpace(domain)
	if domain == "" || channelID == "" || ts == "" {
		return ""
	}
	p := permalinkMessagePath(channelID, ts)
	if p == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.slack.com/archives/%s/%s", domain, channelID, p)
}

// BuildChannelPermalink builds a channel-level archives URL (no message anchor).
func BuildChannelPermalink(teamDomain, channelID string) string {
	domain := strings.TrimSpace(teamDomain)
	domain = strings.TrimSuffix(domain, ".slack.com")
	domain = strings.TrimSpace(domain)
	if domain == "" || channelID == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.slack.com/archives/%s", domain, channelID)
}

func permalinkMessagePath(channelID, ts string) string {
	compact := strings.ReplaceAll(ts, ".", "")
	if len(compact) < 2 {
		return ""
	}
	// Slack permalinks use "p" + ts without the dot between seconds and usec.
	return "p" + compact
}
