package context

import (
	"context"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sessionInfo holds auth.test data cached for one approval flow.
type sessionInfo struct {
	TeamDomain string
	UserID     string
}

type sessionCache struct {
	info *sessionInfo
}

// SessionCache caches auth.test for a single ResolveResourceDetails / context build.
type SessionCache = sessionCache

func (c *sessionCache) get() *sessionInfo {
	if c == nil {
		return nil
	}
	return c.info
}

func (c *sessionCache) set(info *sessionInfo) {
	if c == nil || info == nil {
		return
	}
	c.info = info
}

// resolveSession calls auth.test once per cache and returns team subdomain and authed user ID.
func resolveSession(ctx context.Context, api SlackAPI, creds connectors.Credentials, cache *sessionCache) (*sessionInfo, error) {
	if cache != nil {
		if s := cache.get(); s != nil && s.TeamDomain != "" && s.UserID != "" {
			return s, nil
		}
	}
	var resp authTestResponse
	if err := api.Post(ctx, "auth.test", creds, struct{}{}, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, mapSlackErr(resp.Error)
	}
	domain, err := teamDomainFromAuthURL(resp.URL)
	if err != nil {
		return nil, err
	}
	if resp.UserID == "" {
		return nil, &connectors.ExternalError{StatusCode: 200, Message: "Slack auth.test returned ok=true but no user_id"}
	}
	info := &sessionInfo{TeamDomain: domain, UserID: resp.UserID}
	if cache != nil {
		cache.set(info)
	}
	return info, nil
}

func teamDomainFromAuthURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", &connectors.ExternalError{StatusCode: 200, Message: "Slack auth.test url could not be parsed"}
	}
	host := u.Hostname()
	host = strings.TrimSuffix(host, ".slack.com")
	if host == "" {
		return "", &connectors.ExternalError{StatusCode: 200, Message: "Slack auth.test url missing team subdomain"}
	}
	return host, nil
}
