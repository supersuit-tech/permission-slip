package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// panicCaptureError is swapped in tests to verify Sentry reporting.
var panicCaptureError = CaptureError

// PanicRecoverMiddleware recovers from panics in downstream handlers and
// writes a structured JSON 500 response. It must run inside sentryhttp.Handler
// (so a Sentry hub exists) but before the inner handler chain so panics are
// handled before sentry-go's recover fires.
//
// If the response writer has already sent headers, it only logs and reports
// to Sentry — it does not write a second response.
func PanicRecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &panicRecoverWriter{ResponseWriter: w}
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			traceID := TraceID(r.Context())
			stack := debug.Stack()
			slog.ErrorContext(r.Context(), "http handler panic",
				slog.Any("panic", v),
				slog.String("trace_id", traceID),
				slog.String("stack", string(stack)),
			)
			panicCaptureError(r.Context(), fmt.Errorf("panic: %v\n%s", v, stack))
			if rw.wroteHeader {
				return
			}
			resp := newErrorResponse(ErrInternalPanic, "An unexpected error occurred. Please try again later.", false)
			RespondError(rw, r, http.StatusInternalServerError, resp)
		}()
		next.ServeHTTP(rw, r)
	})
}

type panicRecoverWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (pr *panicRecoverWriter) WriteHeader(code int) {
	if !pr.wroteHeader {
		pr.wroteHeader = true
	}
	pr.ResponseWriter.WriteHeader(code)
}

func (pr *panicRecoverWriter) Write(b []byte) (int, error) {
	if !pr.wroteHeader {
		// First Write sends implicit 200 OK and commits headers.
		pr.wroteHeader = true
	}
	return pr.ResponseWriter.Write(b)
}
