package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// statusRecorder wraps http.ResponseWriter to capture the HTTP status code.
// It implements Unwrap() so http.ResponseController (Go 1.20+) can access
// the underlying writer for Flush, SetWriteDeadline, etc.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.wroteHeader {
		sr.status = code
		sr.wroteHeader = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

// Unwrap returns the underlying ResponseWriter, required by
// http.ResponseController to access Flusher/Hijacker interfaces.
func (sr *statusRecorder) Unwrap() http.ResponseWriter {
	return sr.ResponseWriter
}

// RequestLoggerMiddleware logs structured JSON for every request.
// It records method, path, status, duration, client_ip, and trace_id.
// The trustedProxyHeader is used to extract the real client IP when behind
// a reverse proxy (same header used by the rate limiter).
func RequestLoggerMiddleware(logger *slog.Logger, trustedProxyHeader string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			duration := time.Since(start)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", redactAccessToken(r.RequestURI)),
				slog.Int("status", rec.status),
				slog.Duration("duration", duration),
				slog.String("client_ip", clientIP(r, trustedProxyHeader)),
			}

			if traceID := TraceID(r.Context()); traceID != "" {
				attrs = append(attrs, slog.String("trace_id", traceID))
			}

			// Choose log level based on status code.
			level := slog.LevelInfo
			if rec.status >= 500 {
				level = slog.LevelError
			} else if rec.status >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(r.Context(), level, "http request", attrs...)
		})
	}
}

// redactAccessToken strips the access_token query parameter value from a
// request URI to prevent session tokens from appearing in logs.
// Per RFC 6750 §5.3, logged URIs must not contain bearer tokens.
func redactAccessToken(uri string) string {
	// Fast path: no query string or no access_token param.
	qIdx := strings.IndexByte(uri, '?')
	if qIdx < 0 {
		return uri
	}
	query := uri[qIdx+1:]
	if !strings.Contains(query, "access_token=") {
		return uri
	}

	// Rebuild query, redacting the access_token value.
	var b strings.Builder
	b.WriteString(uri[:qIdx+1]) // path + '?'
	first := true
	for query != "" {
		var param string
		if i := strings.IndexByte(query, '&'); i >= 0 {
			param, query = query[:i], query[i+1:]
		} else {
			param, query = query, ""
		}
		if !first {
			b.WriteByte('&')
		}
		first = false
		if strings.HasPrefix(param, "access_token=") {
			b.WriteString("access_token=[REDACTED]")
		} else {
			b.WriteString(param)
		}
	}
	return b.String()
}
