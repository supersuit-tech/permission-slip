package connectors

import (
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestValidatePrivateNetworkURL_Allows(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		switch host {
		case "tailscale-host.example":
			return []net.IP{net.IPv4(100, 64, 0, 5)}, nil
		case "lan-host.example":
			return []net.IP{net.IPv4(192, 168, 1, 10)}, nil
		default:
			return nil, fmt.Errorf("unexpected host %q", host)
		}
	})

	cases := []string{
		"http://127.0.0.1:18789/hooks",
		"http://[::1]:18789/hooks/wake",
		"http://192.168.0.5/hooks",
		"http://10.0.0.2:18789/hooks",
		"http://100.64.0.5/hooks",
		"http://tailscale-host.example/hooks",
		"http://lan-host.example/hooks",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if err := ValidatePrivateNetworkURL(raw, "webhook_url"); err != nil {
				t.Fatalf("ValidatePrivateNetworkURL(%q) = %v, want nil", raw, err)
			}
		})
	}
}

func TestValidatePrivateNetworkURL_RejectsPublic(t *testing.T) {
	stubLookupIP(t, func(host string) ([]net.IP, error) {
		return []net.IP{net.IPv4(8, 8, 8, 8)}, nil
	})

	cases := []struct {
		raw     string
		wantSub string
	}{
		{"", "must include a host"},
		{"ftp://127.0.0.1/hooks", "http or https"},
		{"http:///hooks", "must include a host"},
		{"http://8.8.8.8/hooks", "public"},
		{"http://public.example/hooks", "public"},
		{"http://169.254.169.254/hooks", "metadata"},
		{"http://metadata.google.internal/hooks", "private network"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			err := ValidatePrivateNetworkURL(tc.raw, "webhook_url")
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}
