package connectors

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidatePrivateNetworkURL validates a webhook URL that must target a private
// network address only (RFC1918, CGNAT/Tailscale 100.64.0.0/10, loopback).
// Plain http is allowed. Public addresses are rejected.
func ValidatePrivateNetworkURL(raw, fieldName string) error {
	if raw == "" {
		return &ValidationError{Message: fmt.Sprintf("%s must include a host", fieldName)}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("%s is not a valid URL: %v", fieldName, err)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return &ValidationError{Message: fmt.Sprintf("%s must use http or https scheme (got %q)", fieldName, u.Scheme)}
	}
	host := u.Hostname()
	if host == "" {
		return &ValidationError{Message: fmt.Sprintf("%s must include a host", fieldName)}
	}

	if ip := net.ParseIP(host); ip != nil {
		if reason := disallowedPrivateTargetReason(ip); reason != "" {
			return &ValidationError{Message: fmt.Sprintf("%s must target a private network address (%s: %s)", fieldName, reason, ip.String())}
		}
		return nil
	}

	lower := strings.ToLower(host)
	if lower == "metadata.google.internal" {
		return &ValidationError{Message: fmt.Sprintf("%s must target a private network address, not %q", fieldName, host)}
	}

	ips, err := lookupIP(host)
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("%s could not be resolved: %v", fieldName, err)}
	}
	if len(ips) == 0 {
		return &ValidationError{Message: fmt.Sprintf("%s resolved to no addresses", fieldName)}
	}
	for _, ip := range ips {
		if reason := disallowedPrivateTargetReason(ip); reason != "" {
			return &ValidationError{Message: fmt.Sprintf("%s resolves to %s (%s) which is not a private network address", fieldName, ip.String(), reason)}
		}
	}
	return nil
}

// disallowedPrivateTargetReason returns a non-empty reason when ip is not an
// allowed private-network target for agent wake webhooks.
func disallowedPrivateTargetReason(ip net.IP) string {
	if ip == nil {
		return "invalid IP"
	}
	if ip.Equal(cloudMetadataIPv4) {
		return "cloud metadata endpoint"
	}
	if isAllowedPrivateNetworkIP(ip) {
		return ""
	}
	if ip.IsMulticast() {
		return "multicast address"
	}
	if ip.IsUnspecified() {
		return "unspecified address"
	}
	return "public address"
}

// isAllowedPrivateNetworkIP reports whether ip is loopback, RFC1918, link-local,
// or CGNAT/Tailscale (100.64.0.0/10).
func isAllowedPrivateNetworkIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127
	}
	return false
}
