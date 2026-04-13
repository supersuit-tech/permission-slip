package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// supabaseProxy returns an http.Handler that reverse-proxies requests from
// /supabase/* to the given Supabase URL, stripping the /supabase prefix.
//
// This allows the frontend to reach Supabase Auth through the same origin as
// the app itself, eliminating the need for CORS configuration and extra port
// exposure. It is especially useful for self-hosted deployments where the
// Supabase stack runs alongside the app (e.g., via `supabase start` on a
// Raspberry Pi).
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

	return http.StripPrefix("/supabase", proxy)
}
