package connectors

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ValidateExternalURL validates a user-supplied URL destined for delivery to or
// by an external service (e.g. a GitHub webhook delivery endpoint, a Shopify
// tracking URL). It enforces the following rules:
//
//   - Scheme must be "https" (webhooks and public endpoints MUST be encrypted).
//   - Host must be non-empty.
//   - Host must not be an IP literal pointing at loopback, private (RFC1918),
//     link-local, multicast, or the cloud metadata endpoint (169.254.169.254).
//   - When the host is a DNS name, every resolved IP must satisfy the same
//     rules. This blocks hostnames that resolve to internal ranges — a common
//     SSRF vector (e.g. internal.mycompany.com → 10.0.0.1).
//
// DNS resolution is performed at validation time as defense-in-depth. It does
// not prevent DNS rebinding between validation and request — for that, the
// HTTP client would also need to re-validate the resolved IP on connect. But
// rejecting obviously-malicious hostnames at the edge catches the common cases
// and removes the cheap attack surface.
//
// fieldName is prepended to error messages so callers can identify the field.
//
// Returns a *ValidationError suitable for returning from an action's validate()
// method.
func ValidateExternalURL(raw, fieldName string) error {
	if raw == "" {
		return &ValidationError{Message: fmt.Sprintf("%s must include a host", fieldName)}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("%s is not a valid URL: %v", fieldName, err)}
	}
	if u.Scheme != "https" {
		return &ValidationError{Message: fmt.Sprintf("%s must use https scheme (got %q)", fieldName, u.Scheme)}
	}
	host := u.Hostname()
	if host == "" {
		return &ValidationError{Message: fmt.Sprintf("%s must include a host", fieldName)}
	}

	// If the host is an IP literal, validate it directly. Otherwise, resolve
	// the hostname and validate every returned IP.
	if ip := net.ParseIP(host); ip != nil {
		if reason := disallowedIPReason(ip); reason != "" {
			return &ValidationError{Message: fmt.Sprintf("%s must not target %s (got %s)", fieldName, reason, ip.String())}
		}
		return nil
	}

	// Reject obviously-local hostnames up front. These are frequently used to
	// reach the caller's loopback interface without an IP literal.
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || lower == "metadata.google.internal" {
		return &ValidationError{Message: fmt.Sprintf("%s must not target internal host %q", fieldName, host)}
	}

	ips, err := lookupIP(host)
	if err != nil {
		return &ValidationError{Message: fmt.Sprintf("%s could not be resolved: %v", fieldName, err)}
	}
	if len(ips) == 0 {
		return &ValidationError{Message: fmt.Sprintf("%s resolved to no addresses", fieldName)}
	}
	for _, ip := range ips {
		if reason := disallowedIPReason(ip); reason != "" {
			return &ValidationError{Message: fmt.Sprintf("%s resolves to %s (%s) which is not allowed", fieldName, ip.String(), reason)}
		}
	}
	return nil
}

// lookupIP is a package-level variable so tests can stub DNS resolution
// without running a real lookup.
var lookupIP = net.LookupIP

// cloudMetadataIPv4 is the link-local address used by AWS, GCP, Azure, and
// DigitalOcean for instance metadata. It is inside 169.254.0.0/16 (blocked by
// the link-local check), but we call it out explicitly in the error message.
var cloudMetadataIPv4 = net.IPv4(169, 254, 169, 254)

// disallowedIPReason returns a non-empty reason string if the IP is in a
// range that should never be the target of an external URL, or "" if the IP
// is allowed.
func disallowedIPReason(ip net.IP) string {
	if ip == nil {
		return "an invalid IP"
	}
	if ip.Equal(cloudMetadataIPv4) {
		return "the cloud metadata endpoint"
	}
	if ip.IsLoopback() {
		return "a loopback address"
	}
	if ip.IsPrivate() {
		return "a private (RFC1918) address"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return "a link-local address"
	}
	if ip.IsMulticast() {
		return "a multicast address"
	}
	if ip.IsUnspecified() {
		return "the unspecified address"
	}
	if ip.IsInterfaceLocalMulticast() {
		return "an interface-local multicast address"
	}
	return ""
}
