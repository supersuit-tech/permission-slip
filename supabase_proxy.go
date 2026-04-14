package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// allowedSupabasePrefixes restricts which upstream Supabase paths the proxy
// will forward. Anything outside these prefixes returns 404 — this prevents
// the proxy from being used as an open/SSRF-style relay into arbitrary
// upstream paths (cloud Supabase management endpoints, cloud metadata
// services via misconfigured SUPABASE_URL, etc.).
//
// Permission Slip's frontend only uses Supabase Auth, so we only allow
// /auth/v1. Adding more surfaces (rest/v1, storage/v1, realtime/v1,
// functions/v1) is intentionally gated behind code changes so the attack
// surface grows deliberately, not by accident.
var allowedSupabasePrefixes = []string{
	"/auth/v1",
}

// supabaseProxy returns an http.Handler that reverse-proxies requests from
// /supabase/<allowed-prefix>/* to the given Supabase URL, stripping the
// /supabase prefix.
//
// This allows the frontend to reach Supabase Auth through the same origin as
// the app itself, eliminating the need for CORS configuration and extra port
// exposure. It is especially useful for self-hosted deployments where the
// Supabase stack runs alongside the app (e.g., via `supabase start` on a
// Raspberry Pi).
//
// The proxy is deliberately narrow: it only forwards paths matching one of
// allowedSupabasePrefixes. Everything else returns 404. This prevents the
// proxy from acting as an open HTTP forwarder into the Supabase upstream
// (or, if SUPABASE_URL is misconfigured, into internal networks).
//
// When VITE_SUPABASE_URL is set at build time (cloud Supabase deployments),
// the frontend talks to Supabase directly and this proxy is unused.
func supabaseProxy(target *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Preserve the original Director but override Host to match the target.
	// Without this, the reverse proxy sends the request with the original
	// Host header (e.g., "raspberrypi.local:8080"), which can confuse
	// upstream services that route by Host.
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)
		req.Host = target.Host
	}

	stripped := http.StripPrefix("/supabase", proxy)

	// Wrap with a prefix allow-list. The /supabase route catches everything
	// under that path; we check the upstream path (with /supabase stripped)
	// against the allow-list before forwarding.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath := strings.TrimPrefix(r.URL.Path, "/supabase")
		if !isAllowedSupabasePath(upstreamPath) {
			http.NotFound(w, r)
			return
		}
		stripped.ServeHTTP(w, r)
	})
}

// isAllowedSupabasePath reports whether path matches one of the allow-listed
// Supabase upstream path prefixes. Matching is either an exact match on the
// prefix or the prefix followed by "/" — so "/auth/v1" allows "/auth/v1" and
// "/auth/v1/token" but not "/auth/v1other".
func isAllowedSupabasePath(path string) bool {
	for _, prefix := range allowedSupabasePrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}
