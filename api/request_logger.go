package api

import (
	"log/slog"
	"net/http"
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
				slog.String("path", r.RequestURI),
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
