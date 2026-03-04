package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type traceIDKey struct{}

// TraceIDMiddleware generates a unique trace ID for each request and stores it
// in the request context. Use TraceID(ctx) to retrieve it.
func TraceIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := generatePrefixedID("trace_", 16)
		if err != nil {
			log.Printf("TraceIDMiddleware: failed to generate random trace ID, using timestamp fallback: %v", err)
			id = fmt.Sprintf("trace_t%d", time.Now().UnixNano())
		}
		ctx := context.WithValue(r.Context(), traceIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceID returns the trace ID from the request context, or empty string if none.
func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(traceIDKey{}).(string)
	return id
}

// SecurityHeadersMiddleware sets standard security headers on every response.
// extraConnectSrc and extraScriptSrc allow additional origins for the CSP
// connect-src and script-src directives respectively (e.g., the Supabase
// project URL for connect-src, or Cloudflare Insights for script-src). Each
// entry is validated as an http(s) origin (scheme://host); invalid entries
// are logged and skipped to prevent CSP directive injection.
//
// sentryCSPEndpoint, when non-empty, adds a report-uri directive pointing to
// Sentry's CSP reporting endpoint so that CSP violations are captured as
// Sentry events. Obtain the URL from your Sentry project under
// Settings → Security Headers → report-uri.
func SecurityHeadersMiddleware(sentryCSPEndpoint string, extraConnectSrc, extraScriptSrc []string) func(http.Handler) http.Handler {
	connectSrc := "'self' https://*.ingest.sentry.io https://*.ingest.us.sentry.io"
	for _, src := range extraConnectSrc {
		origin := sanitizeCSPOrigin(src)
		if origin != "" {
			connectSrc += " " + origin
		}
	}

	scriptSrc := "'self'"
	for _, src := range extraScriptSrc {
		origin := sanitizeCSPOrigin(src)
		if origin != "" {
			scriptSrc += " " + origin
		}
	}

	directives := []string{
		"default-src 'self'",
		"script-src " + scriptSrc,
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com",
		"img-src 'self' data:",
		"connect-src " + connectSrc,
		"worker-src 'self'",
		"frame-ancestors 'none'",
	}

	if endpoint := sanitizeCSPReportURI(sentryCSPEndpoint); endpoint != "" {
		directives = append(directives, "report-uri "+endpoint)
	}

	csp := strings.Join(directives, "; ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			h.Set("Content-Security-Policy", csp)
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
			next.ServeHTTP(w, r)
		})
	}
}

// sanitizeCSPReportURI validates a report-uri value for CSP. It must be a
// valid https URL without characters that could inject additional CSP
// directives (semicolons, quotes, newlines). Returns the validated URL
// string or empty string (with a log warning) on failure.
func sanitizeCSPReportURI(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.ContainsAny(raw, ";'\"\n\r") {
		log.Printf("SecurityHeaders: rejecting CSP report-uri with unsafe characters: %q", raw)
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		log.Printf("SecurityHeaders: rejecting invalid CSP report-uri: %q", raw)
		return ""
	}
	if parsed.Scheme != "https" {
		log.Printf("SecurityHeaders: rejecting non-https CSP report-uri: %q", raw)
		return ""
	}
	return raw
}

// sanitizeCSPOrigin validates a string as an http(s) origin suitable for a CSP
// directive. It returns "scheme://host" on success, or empty string (with a
// log warning) if the value is blank, not a valid URL, uses a non-http(s)
// scheme, or contains characters that could inject CSP directives.
func sanitizeCSPOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Reject values containing characters that could break or inject CSP directives.
	if strings.ContainsAny(raw, ";'\"\n\r") {
		log.Printf("SecurityHeaders: rejecting CSP origin with unsafe characters: %q", raw)
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		log.Printf("SecurityHeaders: rejecting invalid CSP origin: %q", raw)
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		log.Printf("SecurityHeaders: rejecting non-http(s) CSP origin: %q", raw)
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
