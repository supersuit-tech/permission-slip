package connectors

import (
	"fmt"
	"net"
	"strings"
	"testing"
)

// stubLookupIP replaces the DNS lookup for the duration of a test.
func stubLookupIP(t *testing.T, fn func(host string) ([]net.IP, error)) {
	t.Helper()
	orig := lookupIP
	lookupIP = fn
	t.Cleanup(func() { lookupIP = orig })
}

func TestValidateExternalURL_Allows(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		// Any caller-reachable public IP.
		return []net.IP{net.IPv4(140, 82, 121, 3)}, nil
	})

	cases := []string{
		"https://example.com/webhook",
		"https://api.example.com:8443/hooks",
		"https://sub.domain.example.com/path?x=1",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateExternalURL(raw, "url"); err != nil {
				t.Errorf("ValidateExternalURL(%q) unexpected error: %v", raw, err)
			}
		})
	}
}

func TestValidateExternalURL_Rejects(t *testing.T) {
	// Default: DNS never called for these cases; stub to fail loudly so we
	// notice if behavior drifts.
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		return nil, fmt.Errorf("lookupIP should not be called for %q", host)
	})

	cases := []struct {
		name    string
		raw     string
		wantSub string
	}{
		{"empty", "", "must include a host"},
		{"http scheme", "http://example.com/", "must use https"},
		{"ftp scheme", "ftp://example.com/", "must use https"},
		{"file scheme", "file:///etc/passwd", "must use https"},
		{"no host", "https:///path", "must include a host"},
		{"loopback literal", "https://127.0.0.1/", "loopback"},
		{"ipv6 loopback", "https://[::1]/", "loopback"},
		{"rfc1918 10/8", "https://10.0.0.1/", "private"},
		{"rfc1918 172.16/12", "https://172.16.0.1/", "private"},
		{"rfc1918 192.168/16", "https://192.168.1.1/", "private"},
		{"link-local ipv4", "https://169.254.1.2/", "link-local"},
		{"metadata endpoint", "https://169.254.169.254/", "metadata"},
		{"ipv6 link-local", "https://[fe80::1]/", "link-local"},
		{"ipv6 private", "https://[fc00::1]/", "private"},
		{"unspecified", "https://0.0.0.0/", "unspecified"},
		{"localhost hostname", "https://localhost/webhook", "internal host"},
		{"localhost subdomain", "https://anything.localhost/x", "internal host"},
		{"gcp metadata host", "https://metadata.google.internal/", "internal host"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateExternalURL(tc.raw, "url")
			if err == nil {
				t.Fatalf("ValidateExternalURL(%q) = nil, want error containing %q", tc.raw, tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("ValidateExternalURL(%q) error = %q, want substring %q", tc.raw, err.Error(), tc.wantSub)
			}
			if !IsValidationError(err) {
				t.Errorf("ValidateExternalURL(%q) error = %T, want *ValidationError", tc.raw, err)
			}
		})
	}
}

// TestValidateExternalURL_RejectsPrivateDNS ensures a hostname that resolves
// to a private IP is rejected. This is the core SSRF defense: an attacker
// cannot bypass the IP literal check by pointing a public DNS name at an
// internal address.
func TestValidateExternalURL_RejectsPrivateDNS(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		// internal.example.com → 10.0.0.5
		return []net.IP{net.IPv4(10, 0, 0, 5)}, nil
	})

	err := ValidateExternalURL("https://internal.example.com/", "url")
	if err == nil {
		t.Fatal("expected error for DNS-resolved private IP, got nil")
	}
	if !strings.Contains(err.Error(), "private") {
		t.Errorf("error = %q, want substring \"private\"", err.Error())
	}
}

// TestValidateExternalURL_RejectsMixedDNS ensures that if *any* resolved IP
// is disallowed, the URL is rejected — not just when all IPs are bad. A DNS
// response with one public IP and one private IP would otherwise allow
// attackers to smuggle internal traffic through the connector.
func TestValidateExternalURL_RejectsMixedDNS(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		return []net.IP{
			net.IPv4(140, 82, 121, 3), // public
			net.IPv4(10, 0, 0, 5),     // private — must cause rejection
		}, nil
	})

	err := ValidateExternalURL("https://mixed.example.com/", "url")
	if err == nil {
		t.Fatal("expected error when any resolved IP is disallowed, got nil")
	}
	if !strings.Contains(err.Error(), "private") {
		t.Errorf("error = %q, want substring \"private\"", err.Error())
	}
}

func TestValidateExternalURL_DNSFailurePropagates(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		return nil, fmt.Errorf("no such host")
	})

	err := ValidateExternalURL("https://does-not-resolve.example.com/", "url")
	if err == nil {
		t.Fatal("expected error when DNS fails, got nil")
	}
	if !strings.Contains(err.Error(), "could not be resolved") {
		t.Errorf("error = %q, want substring \"could not be resolved\"", err.Error())
	}
}

// TestValidateExternalURL_FieldNamePropagates ensures the caller-supplied
// field name is included in the error so end users can tell which parameter
// failed validation.
func TestValidateExternalURL_FieldNamePropagates(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		return []net.IP{net.IPv4(127, 0, 0, 1)}, nil
	})
	err := ValidateExternalURL("https://foo.test/", "webhook_url")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "webhook_url") {
		t.Errorf("error = %q should mention field name \"webhook_url\"", err.Error())
	}
}
