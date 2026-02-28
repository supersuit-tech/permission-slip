package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware("")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		want   string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.want {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.want)
		}
	}

	// CSP should contain the required directives.
	csp := rec.Header().Get("Content-Security-Policy")
	requiredDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data:",
		"connect-src 'self' https://*.ingest.sentry.io",
		"frame-ancestors 'none'",
	}
	for _, d := range requiredDirectives {
		if !strings.Contains(csp, d) {
			t.Errorf("CSP missing directive %q; full CSP: %s", d, csp)
		}
	}
}

func TestSecurityHeadersMiddleware_ExtraConnectSrc(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware("", "https://abc.supabase.co", "https://other.example.com")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "connect-src 'self' https://*.ingest.sentry.io https://abc.supabase.co https://other.example.com") {
		t.Errorf("CSP connect-src missing extra origins; full CSP: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_InvalidExtraConnectSrc(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// All of these should be rejected — none should appear in the CSP.
	malicious := []string{
		`https://evil.com; script-src *`,       // semicolon injection
		"https://evil.com\nscript-src: *",      // newline injection
		`https://evil.com' 'unsafe-inline`,     // single-quote injection
		`ftp://files.example.com`,              // non-http(s) scheme
		`not-a-url`,                            // no scheme or host
		`javascript:alert(1)`,                  // javascript scheme
	}

	handler := SecurityHeadersMiddleware("", malicious...)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	// connect-src should only contain 'self' + Sentry — all malicious entries rejected.
	if !strings.Contains(csp, "connect-src 'self' https://*.ingest.sentry.io; ") {
		t.Errorf("CSP connect-src should be only 'self' + sentry; full CSP: %s", csp)
	}
	for _, m := range malicious {
		if strings.Contains(csp, m) {
			t.Errorf("CSP should not contain rejected value %q; full CSP: %s", m, csp)
		}
	}
	// Ensure no directive injection occurred.
	if strings.Count(csp, "script-src") != 1 {
		t.Errorf("CSP has injected script-src directive; full CSP: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_MixedValidAndInvalid(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware("",
		"https://good.supabase.co",
		`https://evil.com; script-src *`,
		"http://also-good.example.com",
	)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "https://good.supabase.co") {
		t.Errorf("CSP missing valid origin https://good.supabase.co; full CSP: %s", csp)
	}
	if !strings.Contains(csp, "http://also-good.example.com") {
		t.Errorf("CSP missing valid origin http://also-good.example.com; full CSP: %s", csp)
	}
	if strings.Contains(csp, "evil.com") {
		t.Errorf("CSP should not contain rejected origin evil.com; full CSP: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_EmptyExtraConnectSrc(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Empty strings and whitespace-only strings should be ignored.
	handler := SecurityHeadersMiddleware("", "", "  ")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	// connect-src should be "'self' https://*.ingest.sentry.io" with no trailing spaces from empty inputs.
	if !strings.Contains(csp, "connect-src 'self' https://*.ingest.sentry.io; ") {
		t.Errorf("CSP connect-src has incorrect base formatting; full CSP: %s", csp)
	}
	if strings.Contains(csp, "connect-src 'self' https://*.ingest.sentry.io  ") {
		t.Errorf("CSP connect-src has trailing whitespace from empty extra sources; full CSP: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_SentryCSPReportURI(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	endpoint := "https://o0.ingest.sentry.io/api/0/security/?sentry_key=abc"
	handler := SecurityHeadersMiddleware(endpoint)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "report-uri "+endpoint) {
		t.Errorf("CSP missing report-uri directive; full CSP: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_SentryCSPReportURIInjection(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	malicious := []string{
		`https://evil.com; script-src *`,         // semicolon injection
		"https://evil.com\nscript-src: *",        // newline injection
		`https://evil.com' 'unsafe-inline`,       // single-quote injection
		`http://evil.com/report`,                 // non-https
		`not-a-url`,                              // invalid URL
	}

	for _, endpoint := range malicious {
		handler := SecurityHeadersMiddleware(endpoint)(inner)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		csp := rec.Header().Get("Content-Security-Policy")
		if strings.Contains(csp, "report-uri") {
			t.Errorf("CSP should not contain report-uri for malicious endpoint %q; full CSP: %s", endpoint, csp)
		}
	}
}

func TestSecurityHeadersMiddleware_NoReportURIWhenEmpty(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware("")(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if strings.Contains(csp, "report-uri") {
		t.Errorf("CSP should not contain report-uri when endpoint is empty; full CSP: %s", csp)
	}
}

func TestSecurityHeadersMiddleware_HeadersPresentOnAllMethods(t *testing.T) {
	t.Parallel()
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityHeadersMiddleware("")(inner)

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions} {
		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options missing", method)
		}
		if rec.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s: X-Frame-Options missing", method)
		}
		if rec.Header().Get("Content-Security-Policy") == "" {
			t.Errorf("%s: Content-Security-Policy missing", method)
		}
	}
}
