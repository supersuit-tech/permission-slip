package api

import (
	"net/http"
	"slices"
)

// CORSMiddleware returns middleware that enforces CORS policy based on an
// explicit allow-list of origins. Requests without an Origin header (same-origin
// navigations, server-to-server calls) are always passed through.
//
// When allowedOrigins is empty the middleware operates in "same-origin only"
// mode: requests whose Origin matches the server's own origin (derived from
// the Host header and TLS state) are allowed; everything else is rejected.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// No Origin header → same-origin or non-browser request; let it through.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			// When no explicit allow-list is configured, treat the middleware as
			// "same-origin only": allow requests where Origin matches this server's
			// own origin, and reject other (cross-origin) requests.
			if len(allowedOrigins) == 0 {
				scheme := "http"
				if r.TLS != nil {
					scheme = "https"
				}
				requestOrigin := scheme + "://" + r.Host
				if origin != requestOrigin {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}
			} else if !slices.Contains(allowedOrigins, origin) {
				http.Error(w, "CORS origin not allowed", http.StatusForbidden)
				return
			}

			// Origin is allowed — set CORS response headers.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")

			// Handle CORS preflight. A true preflight includes
			// Access-Control-Request-Method; plain OPTIONS requests are forwarded
			// to the next handler so application-level OPTIONS endpoints still work.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Gateway-Secret")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
