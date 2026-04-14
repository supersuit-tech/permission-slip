package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestIsAllowedSupabasePath(t *testing.T) {
	tests := []struct {
		path    string
		allowed bool
	}{
		// /auth/v1 surface — allowed
		{"/auth/v1", true},
		{"/auth/v1/token", true},
		{"/auth/v1/signup", true},
		{"/auth/v1/.well-known/jwks.json", true},
		{"/auth/v1/factors/enroll", true},

		// Prefix-boundary — must not allow /auth/v1otherthing
		{"/auth/v1other", false},
		{"/auth/v1a", false},

		// Other Supabase surfaces — not allowed (would widen the attack surface)
		{"/rest/v1/users", false},
		{"/storage/v1/object/public/bucket/file", false},
		{"/realtime/v1/websocket", false},
		{"/functions/v1/my-func", false},

		// Management / internal / escape attempts — not allowed
		{"/pg/v1/query", false},
		{"/", false},
		{"", false},
		{"/health", false},
		{"/auth", false},
		{"/auth/", false},

		// Path traversal is a URL parsing concern; these are the raw strings
		// the handler would see after TrimPrefix. Any non-/auth/v1 prefix
		// should be denied.
		{"/../auth/v1/token", false},
		{"/auth//v1/token", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := isAllowedSupabasePath(tc.path)
			if got != tc.allowed {
				t.Errorf("isAllowedSupabasePath(%q) = %v, want %v", tc.path, got, tc.allowed)
			}
		})
	}
}

func TestSupabaseProxy_AllowedPathReaches(t *testing.T) {
	var upstreamHit atomic.Int32
	var lastPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit.Add(1)
		lastPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	handler := supabaseProxy(target)

	req := httptest.NewRequest("GET", "/supabase/auth/v1/token", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Fatalf("allowed path: got status %d, want 200", rw.Code)
	}
	if upstreamHit.Load() != 1 {
		t.Fatalf("allowed path: upstream was not called (hits=%d)", upstreamHit.Load())
	}
	if lastPath != "/auth/v1/token" {
		t.Fatalf("allowed path: upstream saw path %q, want %q", lastPath, "/auth/v1/token")
	}
}

func TestSupabaseProxy_DisallowedPathBlocked(t *testing.T) {
	var upstreamHit atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHit.Add(1)
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	handler := supabaseProxy(target)

	blocked := []string{
		"/supabase/rest/v1/users",
		"/supabase/storage/v1/object/public/bucket/file",
		"/supabase/pg/v1/query",
		"/supabase/",
		"/supabase/health",
		"/supabase/auth/v1other",
	}

	for _, path := range blocked {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rw := httptest.NewRecorder()
			handler.ServeHTTP(rw, req)

			if rw.Code != 404 {
				body, _ := io.ReadAll(rw.Body)
				t.Errorf("disallowed path %q: got status %d, body %q — want 404", path, rw.Code, string(body))
			}
		})
	}

	if upstreamHit.Load() != 0 {
		t.Fatalf("upstream should never be called for disallowed paths (hits=%d)", upstreamHit.Load())
	}
}

func TestSupabaseProxy_HostHeaderRewritten(t *testing.T) {
	// Verify the Director sets req.Host to the upstream target so
	// virtual-host routing works.
	var gotHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		w.WriteHeader(200)
	}))
	defer upstream.Close()

	target, _ := url.Parse(upstream.URL)
	handler := supabaseProxy(target)

	req := httptest.NewRequest("GET", "/supabase/auth/v1/token", nil)
	req.Host = "raspberrypi.local:8080" // incoming host (from the browser)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Fatalf("got status %d, want 200", rw.Code)
	}
	// Upstream Host header should match the target's Host (NOT the incoming Host).
	if !strings.Contains(gotHost, strings.TrimPrefix(upstream.URL, "http://")) {
		t.Errorf("upstream saw Host %q, want target host %q", gotHost, upstream.URL)
	}
}
